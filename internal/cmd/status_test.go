package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStatusCommand_NoArgs(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"status"})

	// when
	err := cmd.Execute()

	// then: should error — repo-path is required
	if err == nil {
		t.Fatal("expected error for no args, got nil")
	}
}

func TestStatusCommand_RejectsTwoArgs(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"status", "arg1", "arg2"})

	// when
	err := cmd.Execute()

	// then: should reject two positional args (max 1)
	if err == nil {
		t.Fatal("expected error for two args, got nil")
	}
}

func TestStatusCommand_TextOutput(t *testing.T) {
	// given: empty repo
	repoDir := t.TempDir()
	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"status", repoDir})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Text goes to stderr
	text := stderr.String()
	if !strings.Contains(text, "paintress status:") {
		t.Errorf("expected stderr to contain 'paintress status:', got:\n%s", text)
	}
	// stdout should be empty for text mode
	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout for text mode, got:\n%s", stdout.String())
	}
}

func TestStatusCommand_JSONOutput(t *testing.T) {
	// given: repo with event files
	repoDir := t.TempDir()
	eventsDir := filepath.Join(repoDir, ".expedition", "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	now := time.Now().UTC().Format(time.RFC3339)
	lines := strings.Join([]string{
		`{"id":"e1","type":"expedition.completed","timestamp":"` + now + `","data":{"expedition":1,"status":"success","issue_id":"PROJ-1"}}`,
		`{"id":"e2","type":"expedition.completed","timestamp":"` + now + `","data":{"expedition":2,"status":"failed","issue_id":"PROJ-2"}}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(eventsDir, today+".jsonl"), []byte(lines+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"status", "-o", "json", repoDir})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if jsonErr := json.Unmarshal(stdout.Bytes(), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", jsonErr, stdout.String())
	}
	if parsed["expeditions"] != float64(2) {
		t.Errorf("expected expeditions=2, got %v", parsed["expeditions"])
	}
	if parsed["successes"] != float64(1) {
		t.Errorf("expected successes=1, got %v", parsed["successes"])
	}
	if parsed["failures"] != float64(1) {
		t.Errorf("expected failures=1, got %v", parsed["failures"])
	}
}

func TestStatusCommand_EmptyRepo(t *testing.T) {
	// given: empty repo directory
	repoDir := t.TempDir()
	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"status", "-o", "json", repoDir})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]any
	if jsonErr := json.Unmarshal(stdout.Bytes(), &parsed); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", jsonErr, stdout.String())
	}
	if parsed["expeditions"] != float64(0) {
		t.Errorf("expected expeditions=0, got %v", parsed["expeditions"])
	}
}
