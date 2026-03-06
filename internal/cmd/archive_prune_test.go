package cmd
// white-box-reason: cobra command construction: NewRootCommand and CLI routing are unexported

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArchivePruneCommand_NoArgs(t *testing.T) {
	// given: no args → falls back to cwd (dry-run by default)
	cmd := NewRootCommand()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"archive-prune"})

	// when
	err := cmd.Execute()

	// then: should succeed using cwd as repo path
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchivePruneCommand_DaysFlagDefault(t *testing.T) {
	// given
	root := NewRootCommand()
	pruneCmd, _, err := root.Find([]string{"archive-prune"})
	if err != nil {
		t.Fatalf("find archive-prune command: %v", err)
	}

	// when
	f := pruneCmd.Flags().Lookup("days")

	// then
	if f == nil {
		t.Fatal("--days flag not found")
	}
	if f.DefValue != "30" {
		t.Errorf("--days default = %q, want %q", f.DefValue, "30")
	}
}

func TestArchivePruneCommand_ExecuteFlagDefault(t *testing.T) {
	// given
	root := NewRootCommand()
	pruneCmd, _, err := root.Find([]string{"archive-prune"})
	if err != nil {
		t.Fatalf("find archive-prune command: %v", err)
	}

	// when
	f := pruneCmd.Flags().Lookup("execute")

	// then
	if f == nil {
		t.Fatal("--execute flag not found")
	}
	if f.DefValue != "false" {
		t.Errorf("--execute default = %q, want %q", f.DefValue, "false")
	}
}

func TestArchivePruneCommand_ShortAliases(t *testing.T) {
	// given
	root := NewRootCommand()
	pruneCmd, _, err := root.Find([]string{"archive-prune"})
	if err != nil {
		t.Fatalf("find archive-prune command: %v", err)
	}

	// then
	aliases := []struct {
		name      string
		shorthand string
	}{
		{"days", "d"},
		{"execute", "x"},
	}

	for _, tc := range aliases {
		f := pruneCmd.Flags().Lookup(tc.name)
		if f == nil {
			t.Errorf("--%s flag not found", tc.name)
			continue
		}
		if f.Shorthand != tc.shorthand {
			t.Errorf("--%s shorthand = %q, want %q", tc.name, f.Shorthand, tc.shorthand)
		}
	}
}

func TestArchivePruneCommand_NegativeDays(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir(), "--days", "-5"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for negative --days, got nil")
	}
}

func TestArchivePruneCommand_ZeroDays(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir(), "--days", "0"})

	// when
	err := cmd.Execute()

	// then
	if err == nil {
		t.Fatal("expected error for zero --days, got nil")
	}
}

func TestArchivePruneCommand_DryRunText(t *testing.T) {
	// given: temp dir with no archive → no candidates
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir()})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestArchivePruneCommand_TextOutput_StdoutClean(t *testing.T) {
	// given: temp dir with no candidates — "No files older" message
	root := NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"archive-prune", t.TempDir()})

	// when
	err := root.Execute()

	// then — text mode: stdout must be empty (all output to stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outBuf.Len() != 0 {
		t.Errorf("text mode should not write to stdout, got: %q", outBuf.String())
	}
	if !strings.Contains(errBuf.String(), "No files older") {
		t.Errorf("expected 'No files older' in stderr, got: %q", errBuf.String())
	}
}

func TestArchivePruneCommand_TextOutput_WithCandidates_StdoutClean(t *testing.T) {
	// given: repo with expired event files
	repoDir := t.TempDir()
	eventsDir := filepath.Join(repoDir, ".expedition", "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(eventsDir, "2025-12-01.jsonl")
	if err := os.WriteFile(oldFile, []byte(`{"id":"old"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	root := NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"archive-prune", repoDir})

	// when
	err := root.Execute()

	// then — text mode with candidates: stdout must still be empty
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outBuf.Len() != 0 {
		t.Errorf("text mode should not write to stdout, got: %q", outBuf.String())
	}
	if !strings.Contains(errBuf.String(), "dry-run") {
		t.Errorf("expected dry-run message in stderr, got: %q", errBuf.String())
	}
}

func TestArchivePruneCommand_DryRunJSON(t *testing.T) {
	// given
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"archive-prune", t.TempDir(), "--output", "json"})

	// when
	err := cmd.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if out == "" {
		t.Error("expected JSON output, got empty")
	}
}

func TestArchivePruneCommand_PrunesEventFiles(t *testing.T) {
	// given: repo with .expedition/events containing expired and recent files
	repoDir := t.TempDir()
	eventsDir := filepath.Join(repoDir, ".expedition", "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	oldFile := filepath.Join(eventsDir, "2025-12-01.jsonl")
	if err := os.WriteFile(oldFile, []byte(`{"id":"old"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	recentFile := filepath.Join(eventsDir, "2026-02-28.jsonl")
	if err := os.WriteFile(recentFile, []byte(`{"id":"recent"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	root := NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"archive-prune", repoDir, "--execute", "--yes"})

	// when
	err := root.Execute()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(oldFile); !os.IsNotExist(statErr) {
		t.Error("expected old event file to be deleted")
	}
	if _, statErr := os.Stat(recentFile); statErr != nil {
		t.Error("expected recent event file to remain")
	}
	output := errBuf.String()
	if !strings.Contains(output, "event") {
		t.Errorf("expected output to mention events, got: %q", output)
	}
}

func TestArchivePruneCommand_EventOnlyPrune(t *testing.T) {
	// given: repo with NO archive candidates but expired event files
	// This tests the codex-found bug: archive candidates=0 must NOT block event pruning
	repoDir := t.TempDir()
	eventsDir := filepath.Join(repoDir, ".expedition", "events")
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	oldEventFile := filepath.Join(eventsDir, "2025-11-01.jsonl")
	if err := os.WriteFile(oldEventFile, []byte(`{"id":"old"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-40 * 24 * time.Hour)
	if err := os.Chtimes(oldEventFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	root := NewRootCommand()
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs([]string{"archive-prune", repoDir, "--execute", "--yes"})

	// when
	err := root.Execute()

	// then — event pruning must fire even with 0 archive candidates
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(oldEventFile); !os.IsNotExist(statErr) {
		t.Error("expected event file to be pruned even with no archive candidates")
	}
}
