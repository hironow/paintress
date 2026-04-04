package domain
// white-box-reason: tests unexported domain functions (parseProvider, excludeIssuesByLabel, etc.)

import (
	"testing"
	"time"
)

func TestParseProvider_ValidProviders(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  Provider
	}{
		{"claude-code", ProviderClaudeCode},
		{"codex", ProviderCodex},
		{"copilot", ProviderCopilot},
		{"gemini-cli", ProviderGeminiCLI},
		{"pi", ProviderPi},
		{"kiro", ProviderKiro},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got, err := parseProvider(tc.input)
			if err != nil {
				t.Fatalf("parseProvider(%q) error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("parseProvider(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseProvider_InvalidProvider(t *testing.T) {
	t.Parallel()
	_, err := parseProvider("unknown-tool")
	if err == nil {
		t.Fatal("parseProvider(unknown-tool) should return error")
	}
}

func TestNewCodingSessionRecord(t *testing.T) {
	t.Parallel()
	rec := NewCodingSessionRecord(ProviderClaudeCode, "opus", "/tmp/repo")

	if rec.ID == "" {
		t.Fatal("ID should be non-empty")
	}
	if rec.Provider != ProviderClaudeCode {
		t.Errorf("Provider = %q, want %q", rec.Provider, ProviderClaudeCode)
	}
	if rec.Status != SessionRunning {
		t.Errorf("Status = %q, want %q", rec.Status, SessionRunning)
	}
	if rec.Model != "opus" {
		t.Errorf("Model = %q, want %q", rec.Model, "opus")
	}
	if rec.WorkDir != "/tmp/repo" {
		t.Errorf("WorkDir = %q, want %q", rec.WorkDir, "/tmp/repo")
	}
	if rec.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be non-zero")
	}
	if rec.ProviderSessionID != "" {
		t.Errorf("ProviderSessionID should be empty initially, got %q", rec.ProviderSessionID)
	}
}

func TestCodingSessionRecord_Complete(t *testing.T) {
	t.Parallel()
	rec := NewCodingSessionRecord(ProviderClaudeCode, "opus", "/tmp/repo")
	before := rec.UpdatedAt

	time.Sleep(time.Millisecond) // ensure time advances
	err := rec.Complete("session-abc-123")
	if err != nil {
		t.Fatalf("Complete() error: %v", err)
	}

	if rec.Status != SessionCompleted {
		t.Errorf("Status = %q, want %q", rec.Status, SessionCompleted)
	}
	if rec.ProviderSessionID != "session-abc-123" {
		t.Errorf("ProviderSessionID = %q, want %q", rec.ProviderSessionID, "session-abc-123")
	}
	if !rec.UpdatedAt.After(before) {
		t.Error("UpdatedAt should advance after Complete()")
	}
}

func TestCodingSessionRecord_Fail(t *testing.T) {
	t.Parallel()
	rec := NewCodingSessionRecord(ProviderClaudeCode, "opus", "/tmp/repo")

	err := rec.Fail("timeout")
	if err != nil {
		t.Fatalf("Fail() error: %v", err)
	}

	if rec.Status != SessionFailed {
		t.Errorf("Status = %q, want %q", rec.Status, SessionFailed)
	}
}

func TestCodingSessionRecord_CompleteFromNonRunning(t *testing.T) {
	t.Parallel()
	rec := NewCodingSessionRecord(ProviderClaudeCode, "opus", "/tmp/repo")
	_ = rec.Complete("id1")

	err := rec.Complete("id2")
	if err == nil {
		t.Fatal("Complete() from completed state should return error")
	}
}

func TestCodingSessionRecord_FailFromNonRunning(t *testing.T) {
	t.Parallel()
	rec := NewCodingSessionRecord(ProviderClaudeCode, "opus", "/tmp/repo")
	_ = rec.Fail("reason")

	err := rec.Fail("again")
	if err == nil {
		t.Fatal("Fail() from failed state should return error")
	}
}
