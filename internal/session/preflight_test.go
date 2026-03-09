package session_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/session"
)

func TestPreflightCheck_ExistingBinary(t *testing.T) {
	// given: "go" should always exist in test environment
	// when
	err := session.PreflightCheck("go")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreflightCheck_MissingBinary(t *testing.T) {
	// given: a binary that should not exist
	// when
	err := session.PreflightCheck("nonexistent-binary-xyz-123")

	// then
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "not found in PATH") {
		t.Errorf("expected 'not found in PATH' in error, got: %v", err)
	}
}

func TestPreflightCheck_MultipleBinaries(t *testing.T) {
	// given: first binary exists, second does not
	// when
	err := session.PreflightCheck("go", "nonexistent-binary-xyz-123")

	// then: should fail on the missing binary
	if err == nil {
		t.Fatal("expected error for missing binary")
	}
	if !strings.Contains(err.Error(), "nonexistent-binary-xyz-123") {
		t.Errorf("expected binary name in error, got: %v", err)
	}
}

func TestPreflightCheck_NoBinaries(t *testing.T) {
	// given: no binaries to check
	// when
	err := session.PreflightCheck()

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- PreflightCheckRemote ---

func TestPreflightCheckRemote_WithRemote(t *testing.T) {
	// given: a git repo with a remote configured
	repoDir := t.TempDir()
	run(t, repoDir, "git", "init")
	run(t, repoDir, "git", "remote", "add", "origin", "https://github.com/example/repo.git")

	// when
	err := session.PreflightCheckRemote(repoDir)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPreflightCheckRemote_NoRemote(t *testing.T) {
	// given: a git repo with no remote
	repoDir := t.TempDir()
	run(t, repoDir, "git", "init")

	// when
	err := session.PreflightCheckRemote(repoDir)

	// then
	if err == nil {
		t.Fatal("expected error for repo without remote")
	}
	if !strings.Contains(err.Error(), "remote") {
		t.Errorf("expected 'remote' in error message, got: %v", err)
	}
	// Error message should mention Pull Request / PR reason
	if !strings.Contains(err.Error(), "Pull Request") && !strings.Contains(err.Error(), "PR") {
		t.Errorf("expected error to mention Pull Request, got: %v", err)
	}
}

func TestPreflightCheckRemote_NotAGitRepo(t *testing.T) {
	// given: a directory that is not a git repo
	dir := t.TempDir()

	// when
	err := session.PreflightCheckRemote(dir)

	// then
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestPreflightCheckRemote_NonexistentDir(t *testing.T) {
	// given: a directory that does not exist
	dir := filepath.Join(t.TempDir(), "nonexistent")

	// when
	err := session.PreflightCheckRemote(dir)

	// then
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestPreflightCheckRemote_MultipleRemotes(t *testing.T) {
	// given: a git repo with multiple remotes
	repoDir := t.TempDir()
	run(t, repoDir, "git", "init")
	run(t, repoDir, "git", "remote", "add", "origin", "https://github.com/example/repo.git")
	run(t, repoDir, "git", "remote", "add", "upstream", "https://github.com/upstream/repo.git")

	// when
	err := session.PreflightCheckRemote(repoDir)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// run executes a command in the given directory, failing the test on error.
func run(t *testing.T, dir, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, out)
	}
}
