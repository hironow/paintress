package session_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/session"
)

// refs issue 0031: D-Mail emission via the transactional outbox.
// paintress produces report (dmail-sendable manifest).

func decodeDMailToolJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(raw), &resp); err != nil {
		t.Fatalf("decode response: %v (raw=%q)", err, string(raw))
	}
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %v", resp)
	}
	content := result["content"].([]any)
	first, _ := content[0].(map[string]any)
	text, _ := first["text"].(string)
	var body map[string]any
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		t.Fatalf("decode tool body: %v (text=%q)", err, text)
	}
	return body
}

func TestMCPServer_DMail_StagesAndFlushesToOutbox(t *testing.T) {
	// given
	continent := t.TempDir()
	req := `{"jsonrpc":"2.0","id":50,"method":"tools/call","params":{"name":"dmail","arguments":{"kind":"report","name":"pt-report-x1-001","description":"Expedition 1 completed X-1","body":"# Report\n\nimplemented + PR opened","issues":["X-1"]}}}` + "\n"
	var out bytes.Buffer
	srv := session.NewMCPServer(strings.NewReader(req), &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	body := decodeDMailToolJSON(t, out.Bytes())
	if body["sent"] != true || body["persistence"] != "transactional-outbox" {
		t.Fatalf("dmail response mismatch: %v", body)
	}
	for _, sub := range []string{"outbox", "archive"} {
		path := filepath.Join(continent, ".expedition", sub, "pt-report-x1-001.md")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("flushed file missing in %s: %v", sub, err)
		}
		if !strings.Contains(string(data), "kind: report") {
			t.Errorf("flushed %s content mismatch:\n%s", sub, string(data))
		}
	}
}

func TestMCPServer_DMail_RejectsKindOutsideProducesSet(t *testing.T) {
	// given: specification is a valid kind but paintress does not produce it
	continent := t.TempDir()
	req := `{"jsonrpc":"2.0","id":51,"method":"tools/call","params":{"name":"dmail","arguments":{"kind":"specification","name":"pt-bad","description":"d","body":"b"}}}` + "\n"
	var out bytes.Buffer
	srv := session.NewMCPServer(strings.NewReader(req), &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	body := decodeDMailToolJSON(t, out.Bytes())
	if body["sent"] != false {
		t.Fatalf("sent = %v, want false for non-produced kind", body["sent"])
	}
	if reason, _ := body["reason"].(string); !strings.Contains(reason, "produce") {
		t.Errorf("reason = %v, want produces-set explanation", body["reason"])
	}
}

func TestMCPServer_DMail_RejectsInvalidPayload(t *testing.T) {
	// given: missing description
	continent := t.TempDir()
	req := `{"jsonrpc":"2.0","id":52,"method":"tools/call","params":{"name":"dmail","arguments":{"kind":"report","name":"pt-no-desc","body":"b"}}}` + "\n"
	var out bytes.Buffer
	srv := session.NewMCPServer(strings.NewReader(req), &out, nil).WithContinent(continent)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	body := decodeDMailToolJSON(t, out.Bytes())
	if body["sent"] != false {
		t.Errorf("sent = %v, want false for invalid payload", body["sent"])
	}
}

func TestMCPServer_DMail_UninitializedWithoutContinent(t *testing.T) {
	// given
	req := `{"jsonrpc":"2.0","id":53,"method":"tools/call","params":{"name":"dmail","arguments":{"kind":"report","name":"x","description":"d","body":"b"}}}` + "\n"
	var out bytes.Buffer
	srv := session.NewMCPServer(strings.NewReader(req), &out, nil)

	// when
	if err := srv.Serve(context.Background()); err != nil {
		t.Fatalf("Serve: %v", err)
	}

	// then
	body := decodeDMailToolJSON(t, out.Bytes())
	if body["initialized"] != false {
		t.Errorf("initialized = %v, want false without continent", body["initialized"])
	}
}
