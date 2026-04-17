package domain
// white-box-reason: tests unexported domain functions (parseProvider, excludeIssuesByLabel, etc.)

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTruncateField_NoTruncation(t *testing.T) {
	t.Parallel()
	s := "short string"
	got, truncated := TruncateField(s, 100)
	if truncated {
		t.Error("should not truncate")
	}
	if got != s {
		t.Errorf("got %q, want %q", got, s)
	}
}

func TestTruncateField_ExactBoundary(t *testing.T) {
	t.Parallel()
	s := "abcd"
	got, truncated := TruncateField(s, 4)
	if truncated {
		t.Error("should not truncate at exact boundary")
	}
	if got != "abcd" {
		t.Errorf("got %q, want %q", got, "abcd")
	}
}

func TestTruncateField_Truncates(t *testing.T) {
	t.Parallel()
	s := strings.Repeat("x", 5000)
	got, truncated := TruncateField(s, 4096)
	if !truncated {
		t.Error("should truncate")
	}
	if len(got) > 4096 {
		t.Errorf("truncated length %d exceeds max 4096", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Error("should end with ...")
	}
}

func TestTruncateField_UTF8Boundary(t *testing.T) {
	t.Parallel()
	// Build a string with multi-byte characters that straddles the boundary.
	// Japanese char = 3 bytes (0xE3 0x81 0x82).
	s := strings.Repeat("\xe3\x81\x82", 2000) // 6000 bytes
	got, truncated := TruncateField(s, 4096)
	if !truncated {
		t.Error("should truncate")
	}
	if len(got) > 4096 {
		t.Errorf("truncated length %d exceeds max 4096", len(got))
	}
	// Should be valid UTF-8 (no broken multi-byte chars).
	withoutSuffix := strings.TrimSuffix(got, "...")
	for i := 0; i < len(withoutSuffix); {
		_, size := []rune(withoutSuffix[i:])[0], len(string([]rune(withoutSuffix[i:])[0]))
		if size == 0 {
			t.Fatal("invalid UTF-8 in truncated output")
		}
		i += size
	}
}

func TestTruncateField_TinyMax(t *testing.T) {
	t.Parallel()
	got, truncated := TruncateField("hello", 2)
	if !truncated {
		t.Error("should truncate")
	}
	if got != "..." {
		t.Errorf("got %q, want %q", got, "...")
	}
}

func TestNewSessionStreamEvent(t *testing.T) {
	t.Parallel()
	data, _ := json.Marshal(map[string]string{"tool_name": "Read"})
	ev := NewSessionStreamEvent("sightjack", ProviderClaudeCode, StreamToolUseStart, data)

	if ev.SchemaVersion != StreamSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", ev.SchemaVersion, StreamSchemaVersion)
	}
	if ev.ID == "" {
		t.Error("ID should be non-empty")
	}
	if ev.Tool != "sightjack" {
		t.Errorf("Tool = %q, want %q", ev.Tool, "sightjack")
	}
	if ev.Provider != ProviderClaudeCode {
		t.Errorf("Provider = %q, want %q", ev.Provider, ProviderClaudeCode)
	}
	if ev.Type != StreamToolUseStart {
		t.Errorf("Type = %q, want %q", ev.Type, StreamToolUseStart)
	}
	if ev.Timestamp.IsZero() {
		t.Error("Timestamp should be non-zero")
	}
}

func TestSessionStreamEvent_WithRaw(t *testing.T) {
	t.Parallel()
	ev := NewSessionStreamEvent("sightjack", ProviderClaudeCode, StreamAssistantText, nil)

	// Short raw -- no truncation.
	ev.WithRaw(`{"type":"assistant"}`)
	if ev.RawTruncated {
		t.Error("should not be truncated")
	}

	// Long raw -- truncation.
	longRaw := strings.Repeat("a", 5000)
	ev.WithRaw(longRaw)
	if !ev.RawTruncated {
		t.Error("should be truncated")
	}
	if len(ev.Raw) > RawFieldMaxBytes {
		t.Errorf("raw length %d exceeds max %d", len(ev.Raw), RawFieldMaxBytes)
	}
}

func TestParseSessionStreamEvent(t *testing.T) {
	t.Parallel()

	valid := NewSessionStreamEvent("sightjack", ProviderClaudeCode, StreamSessionStart, nil)
	if _, err := ParseSessionStreamEvent(valid); err != nil {
		t.Errorf("valid event should pass: %v", err)
	}

	noTool := valid
	noTool.Tool = ""
	if _, err := ParseSessionStreamEvent(noTool); err == nil {
		t.Error("missing tool should fail")
	}

	noType := valid
	noType.Type = ""
	if _, err := ParseSessionStreamEvent(noType); err == nil {
		t.Error("missing type should fail")
	}
}
