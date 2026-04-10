package cmd

// white-box-reason: tests resolveSessionsDir which is an unexported helper used by sessions commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/spf13/cobra"
)

func newTestCmd(withConfig bool) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("path", "", "")
	if withConfig {
		cmd.Flags().String("config", "", "")
	}
	return cmd
}

func TestResolveSessionsDir_PathFlag(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, domain.StateDir), 0755)

	cmd := newTestCmd(false)
	cmd.Flags().Set("path", dir)

	repoRoot, stateDirPath, err := resolveSessionsDir(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoRoot != dir {
		t.Errorf("repoRoot = %q, want %q", repoRoot, dir)
	}
	wantState := filepath.Join(dir, domain.StateDir)
	if stateDirPath != wantState {
		t.Errorf("stateDirPath = %q, want %q", stateDirPath, wantState)
	}
}

func TestResolveSessionsDir_CwdFallback(t *testing.T) {
	dir := t.TempDir()
	// Resolve symlinks to handle macOS /var → /private/var
	dir, _ = filepath.EvalSymlinks(dir)
	os.MkdirAll(filepath.Join(dir, domain.StateDir), 0755)

	oldWd, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(oldWd) })

	cmd := newTestCmd(false)
	repoRoot, _, err := resolveSessionsDir(cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repoRoot != dir {
		t.Errorf("repoRoot = %q, want %q", repoRoot, dir)
	}
}

func TestResolveSessionsDir_MissingStateDir(t *testing.T) {
	dir := t.TempDir() // no StateDir inside

	cmd := newTestCmd(false)
	cmd.Flags().Set("path", dir)

	_, _, err := resolveSessionsDir(cmd)
	if err == nil {
		t.Fatal("expected error for missing state dir")
	}
	if !strings.Contains(err.Error(), "state directory not found:") {
		t.Errorf("error = %q, want 'state directory not found:' prefix", err)
	}
	if !strings.Contains(err.Error(), "run 'paintress init' first") {
		t.Errorf("error = %q, want '(run paintress init first)' suffix", err)
	}
}

func TestResolveSessionsDir_ErrorMessageFormat(t *testing.T) {
	dir := t.TempDir()

	cmd := newTestCmd(false)
	cmd.Flags().Set("path", dir)

	_, _, err := resolveSessionsDir(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
	wantSuffix := "(run 'paintress init' first)"
	if !strings.Contains(err.Error(), wantSuffix) {
		t.Errorf("error %q missing %q", err.Error(), wantSuffix)
	}
}
