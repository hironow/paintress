package cmd_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/cmd"
)

func TestDoctorCommand_NoArgs(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor"})

	// when
	err := root.Execute()

	// then: should succeed or report failed checks (CI has no claude)
	if err != nil && !strings.Contains(err.Error(), "check(s) failed") && !strings.Contains(err.Error(), "checks failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoctorCommand_RejectsTwoArgs(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", "arg1", "arg2"})

	// when
	err := root.Execute()

	// then: should reject two positional args (max 1)
	if err == nil {
		t.Fatal("expected error for two args, got nil")
	}
}

func TestDoctorCommand_OutputFlagDefault(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	root.SetOut(new(bytes.Buffer))
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"doctor"})

	// when
	err := root.Execute()

	// then
	if err != nil && !strings.Contains(err.Error(), "check(s) failed") && !strings.Contains(err.Error(), "checks failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFlag, err := root.PersistentFlags().GetString("output")
	if err != nil {
		t.Fatalf("get output flag: %v", err)
	}
	if outputFlag != "text" {
		t.Errorf("output = %q, want %q", outputFlag, "text")
	}
}

func TestDoctorCommand_OutputFlagJSON(t *testing.T) {
	// given
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"doctor", "--output", "json"})

	// when
	err := root.Execute()

	// then
	if err != nil && !strings.Contains(err.Error(), "check(s) failed") && !strings.Contains(err.Error(), "checks failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	outputFlag, err := root.PersistentFlags().GetString("output")
	if err != nil {
		t.Fatalf("get output flag: %v", err)
	}
	if outputFlag != "json" {
		t.Errorf("output = %q, want %q", outputFlag, "json")
	}
}

func TestDoctorCommand_JSONWithRepoPath(t *testing.T) {
	// given: a temp repo with event files
	repoDir := t.TempDir()
	eventsDir := filepath.Join(repoDir, ".expedition", "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	now := time.Now().UTC().Format(time.RFC3339)
	lines := strings.Join([]string{
		`{"id":"e1","type":"expedition.completed","timestamp":"` + now + `","data":{"status":"success"}}`,
		`{"id":"e2","type":"expedition.completed","timestamp":"` + now + `","data":{"status":"success"}}`,
		`{"id":"e3","type":"expedition.completed","timestamp":"` + now + `","data":{"status":"failed"}}`,
	}, "\n")
	if err := os.WriteFile(filepath.Join(eventsDir, today+".jsonl"), []byte(lines+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"doctor", "-o", "json", repoDir})

	// when
	_ = root.Execute()

	// then
	var output struct {
		Checks  []json.RawMessage `json:"checks"`
		Metrics map[string]any    `json:"metrics"`
	}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if output.Metrics == nil {
		t.Fatal("expected metrics section in JSON output")
	}
	rate, ok := output.Metrics["success_rate"].(string)
	if !ok {
		t.Fatalf("expected string success_rate, got %T", output.Metrics["success_rate"])
	}
	if !strings.Contains(rate, "66.7%") || !strings.Contains(rate, "(2/3)") {
		t.Errorf("unexpected success_rate: %s", rate)
	}
}

func TestDoctorCommand_JSONWithoutRepoPath(t *testing.T) {
	// given: no args → falls back to cwd, metrics are computed from cwd
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"doctor", "-o", "json"})

	// when
	_ = root.Execute()

	// then: should produce valid JSON (metrics may or may not be present depending on cwd)
	var output struct {
		Checks  []json.RawMessage `json:"checks"`
		Metrics map[string]any    `json:"metrics"`
	}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	// With cwd fallback, checks should always be present
	if len(output.Checks) == 0 {
		t.Error("expected at least one check in JSON output")
	}
}

func TestDoctorCommand_JSONWithEmptyRepoPath(t *testing.T) {
	// given: a temp repo with no events
	repoDir := t.TempDir()

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(new(bytes.Buffer))
	root.SetArgs([]string{"doctor", "-o", "json", repoDir})

	// when
	_ = root.Execute()

	// then
	var output struct {
		Checks  []json.RawMessage `json:"checks"`
		Metrics map[string]any    `json:"metrics"`
	}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, buf.String())
	}
	if output.Metrics == nil {
		t.Fatal("expected metrics section even for empty repo")
	}
	rate, ok := output.Metrics["success_rate"].(string)
	if !ok {
		t.Fatalf("expected string success_rate, got %T", output.Metrics["success_rate"])
	}
	if rate != "no events" {
		t.Errorf("expected 'no events', got %q", rate)
	}
}

func TestDoctorCommand_AcceptsOneArg(t *testing.T) {
	// given: doctor now accepts 0 or 1 args
	repoDir := t.TempDir()
	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"doctor", repoDir})

	// when
	err := root.Execute()

	// then: should not error on one arg (failed checks are acceptable in CI (claude may not be available))
	if err != nil && !strings.Contains(err.Error(), "check(s) failed") && !strings.Contains(err.Error(), "checks failed") {
		t.Fatalf("unexpected error for one arg: %v", err)
	}
}
