package session

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// containsStr is a simple substring check without importing strings.
func containsStr(s, sub string) bool {
	return strings.Contains(s, sub)
}

// gitIsolatedEnv returns an environment that strips GIT_DIR and GIT_WORK_TREE
// to prevent test git commands from operating on the parent repo.
func gitIsolatedEnv(dir string) []string {
	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "GIT_DIR=") ||
			strings.HasPrefix(e, "GIT_WORK_TREE=") {
			continue
		}
		env = append(env, e)
	}
	env = append(env, "GIT_CEILING_DIRECTORIES="+filepath.Dir(dir))
	return env
}

// setupGitRepoWithBranch creates a git repo with a test branch.
// Strips GIT_DIR/GIT_WORK_TREE to prevent parent repo corruption.
func setupGitRepoWithBranch(t *testing.T, dir string, branch string) {
	t.Helper()
	gitEnv := gitIsolatedEnv(dir)
	commands := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "test"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "commit", "--allow-empty", "-m", "init"},
		{"git", "checkout", "-b", branch},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = gitEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup (%v) failed: %s", args, string(out))
		}
	}
}

// setupTestRepo creates a minimal git repo for Paintress tests.
// Strips GIT_DIR/GIT_WORK_TREE to prevent parent repo corruption.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitEnv := gitIsolatedEnv(dir)
	commands := [][]string{
		{"git", "init", "-b", "main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "test"},
		{"git", "config", "commit.gpgsign", "false"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		cmd.Env = gitEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git setup (%v) failed: %s", args, string(out))
		}
	}
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	return dir
}

// writeScript creates an executable shell script.
func writeScript(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/bash\n"+content), 0755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
}
