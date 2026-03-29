package session

// white-box-reason: test infrastructure: shared helpers constructing unexported types for sibling tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
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

// ensureExpeditionDirs creates the standard expedition directory structure.
func ensureExpeditionDirs(t *testing.T, continent string) {
	t.Helper()
	for _, dir := range []string{
		domain.ArchiveDir(continent),
		domain.OutboxDir(continent),
		domain.InboxDir(continent),
		filepath.Join(continent, ".expedition", ".run"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
}

// testOutboxStore creates a test outbox store with automatic cleanup.
func testOutboxStore(t *testing.T, continent string) *SQLiteOutboxStore {
	t.Helper()
	store, err := NewOutboxStoreForDir(continent)
	if err != nil {
		t.Fatalf("create outbox store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// failingEmitter is a minimal ExpeditionEventEmitter that always returns an error.
// Duplicated here for package session tests (also exists in dmail_test.go which is session_test).
type failingEmitter struct {
	err error
}

func (f *failingEmitter) EmitStartExpedition(_, _ int, _ string, _ time.Time) error { return f.err }
func (f *failingEmitter) EmitCompleteExpedition(_ int, _, _, _, _, _ string, _ time.Time) error {
	return f.err
}
func (f *failingEmitter) EmitSpecRegistered(_ string, _ []domain.WaveStepDef, _ string, _ time.Time) error {
	return f.err
}
func (f *failingEmitter) EmitInboxReceived(_, _ string, _ time.Time) error { return f.err }
func (f *failingEmitter) EmitGommage(_ int, _ time.Time) error             { return f.err }
func (f *failingEmitter) EmitGradientChange(_ int, _ string, _ time.Time) error {
	return f.err
}
func (f *failingEmitter) EmitRetryAttempted(_ string, _ int, _ time.Time) error { return f.err }
func (f *failingEmitter) EmitEscalated(_ string, _ []string, _ time.Time) error { return f.err }
func (f *failingEmitter) EmitResolved(_ string, _ []string, _ time.Time) error  { return f.err }
func (f *failingEmitter) EmitDMailStaged(_ string, _ time.Time) error           { return f.err }
func (f *failingEmitter) EmitDMailFlushed(_ int, _ time.Time) error             { return f.err }
func (f *failingEmitter) EmitDMailArchived(_ string, _ time.Time) error         { return f.err }
func (f *failingEmitter) EmitGommageRecovery(_ int, _, _ string, _ int, _ string, _ time.Time) error {
	return f.err
}
func (f *failingEmitter) EmitCheckpoint(_ int, _, _ string, _ int, _ time.Time) error { return f.err }

// writeScript creates an executable shell script.
func writeScript(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/bash\n"+content), 0755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
}
