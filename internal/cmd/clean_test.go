package cmd

// white-box-reason: cobra command construction: NewRootCommand and CLI routing are unexported

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanCmd_NothingToClean(t *testing.T) {
	// given: empty directory with no .expedition/
	dir := t.TempDir()

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"clean", "--yes", dir})

	// when
	err := cmd.Execute()

	// then: should succeed with "nothing to clean" message
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := buf.String(); !strings.Contains(got, "Nothing to clean") {
		t.Errorf("expected 'Nothing to clean' in output, got: %s", got)
	}
}

func TestCleanCmd_DeletesExpeditionDir(t *testing.T) {
	// given: .expedition/ directory with config
	dir := t.TempDir()
	expDir := filepath.Join(dir, ".expedition")
	if err := os.MkdirAll(expDir, 0755); err != nil {
		t.Fatalf("create expedition dir: %v", err)
	}
	cfgPath := filepath.Join(expDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("tracker:\n  team: MY\n"), 0644); err != nil {
		t.Fatalf("create config: %v", err)
	}

	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"clean", "--yes", dir})

	// when
	err := cmd.Execute()

	// then: should succeed and delete .expedition/
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(expDir); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected .expedition/ dir to be deleted")
	}
}
