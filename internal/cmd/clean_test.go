package cmd_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/cmd"
)

func TestCleanCmd_NothingToClean(t *testing.T) {
	// given: empty directory with no .expedition/
	dir := t.TempDir()

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"clean", "--yes", dir})

	// when
	err := root.Execute()

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

	root := cmd.NewRootCommand()
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"clean", "--yes", dir})

	// when
	err := root.Execute()

	// then: should succeed and delete .expedition/
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(expDir); !errors.Is(err, fs.ErrNotExist) {
		t.Error("expected .expedition/ dir to be deleted")
	}
}
