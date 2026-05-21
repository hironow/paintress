package session_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestMCPServer_ListsAllPhase1Tools(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: all 4 Phase 1 tools advertised, with stable names
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v (raw=%q)", err, out.String())
	}
	if resp["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", resp)
	}
	tools, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("tools list missing: %v", result["tools"])
	}
	want := map[string]bool{
		"paintress.ping":            false,
		"paintress.next_issue":      false,
		"paintress.update_gradient": false,
		"paintress.append_journal":  false,
	}
	for _, t0 := range tools {
		entry, _ := t0.(map[string]any)
		if name, _ := entry["name"].(string); name != "" {
			want[name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("missing Phase 1 tool: %s", name)
		}
	}
}

func TestMCPServer_CallsPingTool(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"paintress.ping","arguments":{}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v (raw=%q)", err, out.String())
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("content list mismatch: %v", result["content"])
	}
	first, _ := content[0].(map[string]any)
	if first["text"] != "pong" {
		t.Errorf("text = %v, want pong", first["text"])
	}
}

func TestMCPServer_RejectsUnknownTool(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"paintress.does_not_exist","arguments":{}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v (raw=%q)", err, out.String())
	}
	rpcErr, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error, got %v", resp)
	}
	if code, _ := rpcErr["code"].(float64); int(code) != -32601 {
		t.Errorf("error code = %v, want -32601", rpcErr["code"])
	}
}

func TestMCPServer_NextIssue_UninitializedContinent(t *testing.T) {
	// given: NewMCPServer without WithContinent → continent is empty.
	// Real impl must signal "uninitialized" so the claude code session
	// surfaces a clear error rather than acting on stale defaults.
	in := strings.NewReader(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"paintress.next_issue","arguments":{}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	body := decodeFirstText(t, &out)
	if body["initialized"] != false {
		t.Errorf("initialized = %v, want false (empty continent)", body["initialized"])
	}
	if _, ok := body["reason"]; !ok {
		t.Errorf("reason missing: %v", body)
	}
}

func TestMCPServer_NextIssue_RealImpl_EmptyJournal(t *testing.T) {
	// given: temp continent dir with no journal / pr-index yet.
	continent := t.TempDir()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"paintress.next_issue","arguments":{}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: initialized=true, no completed work, next expedition = 1.
	body := decodeFirstText(t, &out)
	if body["initialized"] != true {
		t.Errorf("initialized = %v, want true", body["initialized"])
	}
	if got, _ := body["next_expedition_number"].(float64); int(got) != 1 {
		t.Errorf("next_expedition_number = %v, want 1", body["next_expedition_number"])
	}
	if ids, _ := body["completed_issue_ids"].([]any); len(ids) != 0 {
		t.Errorf("completed_issue_ids = %v, want empty []", body["completed_issue_ids"])
	}
	if body["last_pr"] != nil {
		t.Errorf("last_pr = %v, want nil", body["last_pr"])
	}
}

func TestMCPServer_NextIssue_RealImpl_WithPRIndex(t *testing.T) {
	// given: temp continent with pr-index.jsonl containing 2 entries.
	continent := t.TempDir()
	journalDir := domain.JournalDir(continent)
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("mkdir journal: %v", err)
	}
	prIndex := filepath.Join(journalDir, "pr-index.jsonl")
	prBody := `{"expedition":1,"issue_id":"X-1","pr_url":"https://github.com/example/repo/pull/1"}
{"expedition":2,"issue_id":"X-2","pr_url":"https://github.com/example/repo/pull/2"}
`
	if err := os.WriteFile(prIndex, []byte(prBody), 0o644); err != nil {
		t.Fatalf("write pr-index: %v", err)
	}

	in := strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"paintress.next_issue","arguments":{}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: completed_issue_ids = [X-1, X-2], next_expedition_number = 3,
	// last_pr.issue_id = X-2.
	resp := decodeFirstText(t, &out)
	if resp["initialized"] != true {
		t.Errorf("initialized = %v, want true", resp["initialized"])
	}
	if got, _ := resp["next_expedition_number"].(float64); int(got) != 3 {
		t.Errorf("next_expedition_number = %v, want 3", resp["next_expedition_number"])
	}
	ids, _ := resp["completed_issue_ids"].([]any)
	if len(ids) != 2 || ids[0] != "X-1" || ids[1] != "X-2" {
		t.Errorf("completed_issue_ids = %v, want [X-1, X-2]", resp["completed_issue_ids"])
	}
	lastPR, _ := resp["last_pr"].(map[string]any)
	if lastPR == nil || lastPR["issue_id"] != "X-2" {
		t.Errorf("last_pr.issue_id = %v, want X-2 (got %v)", "X-2", lastPR)
	}
}

func TestMCPServer_UpdateGradientStub_EchoesDelta(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"paintress.update_gradient","arguments":{"delta":3}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: stub echoes the requested delta so the contract is testable
	// end-to-end before the real gradient gauge wiring lands.
	body := decodeFirstText(t, &out)
	if got, _ := body["delta"].(float64); int(got) != 3 {
		t.Errorf("delta = %v, want 3", body["delta"])
	}
	if got, _ := body["new_level"].(float64); int(got) != 3 {
		t.Errorf("new_level = %v, want 3 (stub mirrors delta)", body["new_level"])
	}
}

func TestMCPServer_AppendJournalStub_EchoesEntry(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"paintress.append_journal","arguments":{"expedition":42,"issue_id":"PAI-1","status":"completed"}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: stub echoes the entry untouched.
	body := decodeFirstText(t, &out)
	entry, ok := body["entry"].(map[string]any)
	if !ok {
		t.Fatalf("entry missing: %v", body)
	}
	if entry["issue_id"] != "PAI-1" {
		t.Errorf("issue_id = %v, want PAI-1", entry["issue_id"])
	}
	if entry["status"] != "completed" {
		t.Errorf("status = %v, want completed", entry["status"])
	}
}

// decodeFirstText extracts the JSON payload from the first content
// item of the MCP tools/call response. Stub responses ship a single
// JSON-string text entry so the body is a JSON object inside a string.
func decodeFirstText(t *testing.T, out *bytes.Buffer) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v (raw=%q)", err, out.String())
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("missing content: %v", result)
	}
	first, _ := content[0].(map[string]any)
	text, _ := first["text"].(string)
	var body map[string]any
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		t.Fatalf("decode inner JSON: %v (raw=%q)", err, text)
	}
	return body
}

func TestMCPServer_RejectsUnknownMethod(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":4,"method":"completion/complete"}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode response: %v (raw=%q)", err, out.String())
	}
	rpcErr, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error, got %v", resp)
	}
	if code, _ := rpcErr["code"].(float64); int(code) != -32601 {
		t.Errorf("error code = %v, want -32601", rpcErr["code"])
	}
}
