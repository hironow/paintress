package session_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase/port"
)

// recordingEmitter is a minimal ExpeditionEventEmitter for white-box
// testing of MCP wiring. It records emit calls + appends the event to
// the configured store so the test can assert event store state.
// It does not implement the full ExpeditionEventEmitter contract;
// methods unused by the MCP tools return nil.
type recordingEmitter struct {
	store     port.EventStore
	gradients []domain.GradientChangedData
	completes []domain.ExpeditionCompletedData
}

func (r *recordingEmitter) EmitGradientChange(level int, operator string, now time.Time) error {
	r.gradients = append(r.gradients, domain.GradientChangedData{Level: level, Operator: operator})
	ev, err := domain.NewEvent(domain.EventGradientChanged, domain.GradientChangedData{Level: level, Operator: operator}, now)
	if err != nil {
		return err
	}
	_, err = r.store.Append(context.Background(), ev)
	return err
}

func (r *recordingEmitter) EmitCompleteExpedition(expedition int, status, issueID, bugsFound, waveID, stepID string, now time.Time) error { // nolint: revive
	r.completes = append(r.completes, domain.ExpeditionCompletedData{
		Expedition: expedition,
		Status:     status,
		IssueID:    issueID,
		WaveID:     waveID,
		StepID:     stepID,
		BugsFound:  bugsFound,
	})
	ev, err := domain.NewEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: expedition,
		Status:     status,
		IssueID:    issueID,
		WaveID:     waveID,
		StepID:     stepID,
		BugsFound:  bugsFound,
	}, now)
	if err != nil {
		return err
	}
	_, err = r.store.Append(context.Background(), ev)
	return err
}

// Below: unused ExpeditionEventEmitter methods (Nop satisfies the port).
func (r *recordingEmitter) EmitStartExpedition(_, _ int, _ string, _ time.Time) error {
	return nil
}
func (r *recordingEmitter) EmitSpecRegistered(_ string, _ []domain.WaveStepDef, _ string, _ time.Time) error {
	return nil
}
func (r *recordingEmitter) EmitInboxReceived(_, _ string, _ time.Time) error      { return nil }
func (r *recordingEmitter) EmitGommage(_ int, _ time.Time) error                  { return nil }
func (r *recordingEmitter) EmitRetryAttempted(_ string, _ int, _ time.Time) error { return nil }
func (r *recordingEmitter) EmitEscalated(_ string, _ []string, _ time.Time) error { return nil }
func (r *recordingEmitter) EmitResolved(_ string, _ []string, _ time.Time) error  { return nil }
func (r *recordingEmitter) EmitDMailStaged(_ string, _ time.Time) error           { return nil }
func (r *recordingEmitter) EmitDMailFlushed(_ int, _ time.Time) error             { return nil }
func (r *recordingEmitter) EmitDMailArchived(_ string, _ time.Time) error         { return nil }
func (r *recordingEmitter) EmitGommageRecovery(_ int, _, _ string, _ int, _ string, _ time.Time) error {
	return nil
}
func (r *recordingEmitter) EmitCheckpoint(_ int, _, _ string, _ int, _ time.Time) error {
	return nil
}
func (r *recordingEmitter) SetSeqAllocator(_ port.SeqAllocator) {}

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
		"ping":            false,
		"next_issue":      false,
		"update_gradient": false,
		"append_journal":  false,
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
	if strings.Contains(out.String(), "linear-mcp") {
		t.Errorf("tools list should not mention linear-mcp after MCP pivot: %q", out.String())
	}
}

func TestMCPServer_CallsPingTool(t *testing.T) {
	// given
	in := strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ping","arguments":{}}}` + "\n")
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
	in := strings.NewReader(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"does_not_exist","arguments":{}}}` + "\n")
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
	// Real impl must signal "uninitialized" so the Claude Code session
	// surfaces a clear error rather than acting on stale defaults.
	in := strings.NewReader(`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"next_issue","arguments":{}}}` + "\n")
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
	in := strings.NewReader(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"next_issue","arguments":{}}}` + "\n")
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
	if strings.Contains(out.String(), "linear-mcp") {
		t.Errorf("next_issue response should not mention linear-mcp after MCP pivot: %q", out.String())
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

	in := strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"next_issue","arguments":{}}}` + "\n")
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
	if instruction, _ := resp["instruction"].(string); strings.Contains(instruction, "linear-mcp") {
		t.Errorf("instruction should not mention linear-mcp after MCP pivot: %q", instruction)
	}
}

func TestMCPServer_UpdateGradient_UninitializedContinent(t *testing.T) {
	// given: empty continent → uninitialized response with delta echo.
	in := strings.NewReader(`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"update_gradient","arguments":{"delta":3}}}` + "\n")
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
	if got, _ := body["delta"].(float64); int(got) != 3 {
		t.Errorf("delta = %v, want 3", body["delta"])
	}
}

func TestMCPServer_UpdateGradient_RealImpl_EmptyEventStore(t *testing.T) {
	// given: temp continent dir with no events yet → current_level = 0.
	continent := t.TempDir()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"update_gradient","arguments":{"delta":5}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: initialized=true, current=0, delta=5, preview=5.
	body := decodeFirstText(t, &out)
	if body["initialized"] != true {
		t.Errorf("initialized = %v, want true", body["initialized"])
	}
	if got, _ := body["current_level"].(float64); int(got) != 0 {
		t.Errorf("current_level = %v, want 0", body["current_level"])
	}
	if got, _ := body["preview_level"].(float64); int(got) != 5 {
		t.Errorf("preview_level = %v, want 5", body["preview_level"])
	}
	if body["persistence"] != "preview-only" {
		t.Errorf("persistence = %v, want preview-only", body["persistence"])
	}
}

func TestMCPServer_AppendJournal_UninitializedContinent(t *testing.T) {
	// given: empty continent → uninitialized response.
	in := strings.NewReader(`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"append_journal","arguments":{"expedition":42,"issue_id":"PAI-1","status":"completed"}}}` + "\n")
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
}

func TestMCPServer_AppendJournal_RealImpl_PersistsToFilesystem(t *testing.T) {
	// given: temp continent + minimal valid input.
	continent := t.TempDir()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"append_journal","arguments":{"expedition":42,"issue_id":"PAI-1","issue_title":"Fix login","status":"completed","pr_url":"https://github.com/example/repo/pull/7","reason":"validated"}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: persisted=true + journal file exists + pr-index appended.
	body := decodeFirstText(t, &out)
	if body["persisted"] != true {
		t.Errorf("persisted = %v, want true (body=%v)", body["persisted"], body)
	}
	if got, _ := body["expedition"].(float64); int(got) != 42 {
		t.Errorf("expedition = %v, want 42", body["expedition"])
	}
	if body["issue_id"] != "PAI-1" {
		t.Errorf("issue_id = %v, want PAI-1", body["issue_id"])
	}
	// journal file on disk
	journalDir := domain.JournalDir(continent)
	if _, err := os.Stat(filepath.Join(journalDir, "042.md")); err != nil {
		t.Errorf("journal file 042.md missing: %v", err)
	}
	// pr-index appended
	if body["pr_index_updated"] != true {
		t.Errorf("pr_index_updated = %v, want true", body["pr_index_updated"])
	}
	if _, err := os.Stat(filepath.Join(journalDir, "pr-index.jsonl")); err != nil {
		t.Errorf("pr-index.jsonl missing: %v", err)
	}
}

func TestMCPServer_UpdateGradient_Phase4_EmitsGradientChangedEvent(t *testing.T) {
	// given: temp continent with recording emitter wired.
	continent := t.TempDir()
	stateDir := filepath.Join(continent, domain.StateDir)
	store := session.NewEventStore(stateDir, nil)
	emitter := &recordingEmitter{store: store}
	in := strings.NewReader(`{"jsonrpc":"2.0","id":50,"method":"tools/call","params":{"name":"update_gradient","arguments":{"delta":3}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent).WithEmitter(emitter)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: response reports persistence='event-store' + new_level reflected.
	body := decodeFirstText(t, &out)
	if body["initialized"] != true {
		t.Fatalf("initialized = %v, want true: %v", body["initialized"], body)
	}
	if body["persistence"] != "event-store" {
		t.Errorf("persistence = %v, want event-store", body["persistence"])
	}
	if got, _ := body["new_level"].(float64); int(got) != 3 {
		t.Errorf("new_level = %v, want 3", body["new_level"])
	}
	// event store contains exactly one EventGradientChanged with level=3.
	events, _, err := store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	gradEvents := 0
	for _, ev := range events {
		if ev.Type == domain.EventGradientChanged {
			gradEvents++
			var data domain.GradientChangedData
			if err := json.Unmarshal(ev.Data, &data); err != nil {
				t.Fatalf("decode gradient event: %v", err)
			}
			if data.Level != 3 {
				t.Errorf("gradient event level = %d, want 3", data.Level)
			}
		}
	}
	if gradEvents != 1 {
		t.Errorf("EventGradientChanged count = %d, want 1: %v", gradEvents, events)
	}
}

func TestMCPServer_UpdateGradient_Phase4_PreviewWhenEmitterDisabled(t *testing.T) {
	// given: emitter NOT enabled (default) → preview-only contract retained.
	continent := t.TempDir()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":51,"method":"tools/call","params":{"name":"update_gradient","arguments":{"delta":5}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: persistence=preview-only + no event emitted.
	body := decodeFirstText(t, &out)
	if body["persistence"] != "preview-only" {
		t.Errorf("persistence = %v, want preview-only (emitter disabled)", body["persistence"])
	}
	stateDir := filepath.Join(continent, domain.StateDir)
	store := session.NewEventStore(stateDir, nil)
	events, _, _ := store.LoadAll(context.Background())
	for _, ev := range events {
		if ev.Type == domain.EventGradientChanged {
			t.Errorf("unexpected EventGradientChanged: %v", ev)
		}
	}
}

func TestMCPServer_AppendJournal_Phase4_EmitsExpeditionCompletedEvent(t *testing.T) {
	// given: temp continent with recording emitter wired.
	continent := t.TempDir()
	stateDir := filepath.Join(continent, domain.StateDir)
	store := session.NewEventStore(stateDir, nil)
	emitter := &recordingEmitter{store: store}
	in := strings.NewReader(`{"jsonrpc":"2.0","id":52,"method":"tools/call","params":{"name":"append_journal","arguments":{"expedition":7,"issue_id":"PAI-7","status":"success","pr_url":"https://example/pull/7","wave_id":"w-1","step_id":"s-1"}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent).WithEmitter(emitter)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: persistence='event-store+filesystem' + event emitted.
	body := decodeFirstText(t, &out)
	if body["persisted"] != true {
		t.Fatalf("persisted = %v, want true: %v", body["persisted"], body)
	}
	if body["persistence"] != "event-store+filesystem" {
		t.Errorf("persistence = %v, want event-store+filesystem", body["persistence"])
	}
	events, _, err := store.LoadAll(context.Background())
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	completedCount := 0
	for _, ev := range events {
		if ev.Type == domain.EventExpeditionCompleted {
			completedCount++
			var data domain.ExpeditionCompletedData
			if err := json.Unmarshal(ev.Data, &data); err != nil {
				t.Fatalf("decode expedition completed: %v", err)
			}
			if data.Expedition != 7 || data.IssueID != "PAI-7" || data.Status != "success" {
				t.Errorf("ExpeditionCompletedData mismatch: %+v", data)
			}
		}
	}
	if completedCount != 1 {
		t.Errorf("EventExpeditionCompleted count = %d, want 1", completedCount)
	}
}

func TestMCPServer_AppendJournal_RealImpl_RejectsMissingRequiredFields(t *testing.T) {
	// given: empty issue_id is invalid.
	continent := t.TempDir()
	in := strings.NewReader(`{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"append_journal","arguments":{"expedition":1,"status":"completed"}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	body := decodeFirstText(t, &out)
	if body["persisted"] != false {
		t.Errorf("persisted = %v, want false (missing issue_id)", body["persisted"])
	}
	if _, ok := body["reason"]; !ok {
		t.Errorf("reason missing: %v", body)
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

func TestMCPServer_Initialize_Handshake(t *testing.T) {
	// given: client sends initialize with a different protocol version
	in := strings.NewReader(`{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"claude-code","version":"1.0"}}}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: server returns ITS supported version (not an echo), + tools cap + serverInfo
	var resp struct {
		Result struct {
			ProtocolVersion string                     `json:"protocolVersion"`
			Capabilities    map[string]json.RawMessage `json:"capabilities"`
			Instructions    string                     `json:"instructions"`
			ServerInfo      struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &resp); err != nil {
		t.Fatalf("decode initialize response: %v (raw=%q)", err, out.String())
	}
	if resp.Result.ProtocolVersion != "2024-11-05" {
		t.Errorf("protocolVersion = %q, want 2024-11-05 (server supported, not echo of client 2025-06-18)", resp.Result.ProtocolVersion)
	}
	if _, ok := resp.Result.Capabilities["tools"]; !ok {
		t.Errorf("capabilities.tools missing: %v", resp.Result.Capabilities)
	}
	if resp.Result.ServerInfo.Name != "paintress" {
		t.Errorf("serverInfo.name = %q, want paintress", resp.Result.ServerInfo.Name)
	}
	if !strings.Contains(resp.Result.Instructions, "implementer") {
		t.Errorf("instructions = %q, want a one-paragraph implementer role summary", resp.Result.Instructions)
	}
}

func TestMCPServer_NotificationsInitialized_NoResponse(t *testing.T) {
	// given: a JSON-RPC notification (no id)
	in := strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n")
	var out bytes.Buffer
	srv := session.NewMCPServer(in, &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then: notifications must not produce a response
	if strings.TrimSpace(out.String()) != "" {
		t.Errorf("notification must produce no response, got: %q", out.String())
	}
}
