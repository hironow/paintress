package session

// white-box-reason: session internals: tests unexported localGitExecutor for worktree operations

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// initGitRepoForWorktree creates a temporary git repo initialized with --initial-branch main
// and configured with test user identity. Returns the repo directory path.
func initGitRepoForWorktree(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	executor := &localGitExecutor{}
	ctx := context.Background()
	if _, err := executor.Git(ctx, dir, "init", "--initial-branch", "main"); err != nil {
		t.Fatalf("git init: %v", err)
	}
	if _, err := executor.Git(ctx, dir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email: %v", err)
	}
	if _, err := executor.Git(ctx, dir, "config", "user.name", "Test User"); err != nil {
		t.Fatalf("git config name: %v", err)
	}
	return dir
}

// initGitRepoForWorktreeWithCommit creates a temporary git repo with an initial empty commit.
func initGitRepoForWorktreeWithCommit(t *testing.T) string {
	t.Helper()
	dir := initGitRepoForWorktree(t)
	executor := &localGitExecutor{}
	ctx := context.Background()
	if _, err := executor.Git(ctx, dir, "commit", "--allow-empty", "-m", "init"); err != nil {
		t.Fatalf("git commit: %v", err)
	}
	return dir
}

// === localGitExecutor tests ===

func TestLocalGitExecutor_Git_StatusOnNewRepo(t *testing.T) {
	// given
	dir := t.TempDir()
	executor := &localGitExecutor{}
	ctx := context.Background()

	// init a git repo in the temp dir
	_, err := executor.Git(ctx, dir, "init")
	if err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// when
	out, err := executor.Git(ctx, dir, "status")

	// then
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if !strings.Contains(string(out), "On branch") {
		t.Errorf("expected 'On branch' in output, got: %s", string(out))
	}
}

func TestLocalGitExecutor_Shell_EchoCommand(t *testing.T) {
	// given
	dir := t.TempDir()
	executor := &localGitExecutor{}
	ctx := context.Background()

	// when
	out, err := executor.Shell(ctx, dir, "echo hello")

	// then
	if err != nil {
		t.Fatalf("shell command failed: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "hello" {
		t.Errorf("expected 'hello', got: %q", got)
	}
}

// === localGitExecutor-based tests (formerly container tests) ===

func TestLocalGitExecutor_Git_StatusOnNewRepoWithMainBranch(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	workDir := initGitRepoForWorktree(t)

	// when
	out, err := executor.Git(ctx, workDir, "status")

	// then
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if !strings.Contains(string(out), "On branch main") {
		t.Errorf("expected 'On branch main' in output, got: %s", string(out))
	}
}

func TestLocalGitExecutor_Shell_EchoCommand_InTempDir(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	dir := t.TempDir()

	// when
	out, err := executor.Shell(ctx, dir, "echo hello")

	// then
	if err != nil {
		t.Fatalf("shell command failed: %v", err)
	}
	got := strings.TrimSpace(string(out))
	if got != "hello" {
		t.Errorf("expected 'hello', got: %q", got)
	}
}

func TestWorktreePool_Init_CreatesWorktrees(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 3)

	// when
	err := pool.Init(ctx)

	// then
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// verify: git worktree list shows 4 entries (1 main + 3 workers)
	out, err := executor.Git(ctx, repoDir, "worktree", "list")
	if err != nil {
		t.Fatalf("git worktree list failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 worktree entries (1 main + 3 workers), got %d:\n%s", len(lines), string(out))
	}

	// verify: 3 paths available in workers channel
	if len(pool.workers) != 3 {
		t.Fatalf("expected 3 workers in channel, got %d", len(pool.workers))
	}
	for i := range 3 {
		path := <-pool.workers
		expectedName := fmt.Sprintf("worker-%03d", i+1)
		if !strings.HasSuffix(path, expectedName) {
			t.Errorf("worker %d: expected path ending with %q, got %q", i+1, expectedName, path)
		}
		pool.workers <- path // put back
	}
}

func TestWorktreePool_Init_PrunesStale(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	// create a worktree manually, then remove its directory to simulate a crash
	staleWorktreePath := repoDir + "/stale-wt"
	_, err := executor.Git(ctx, repoDir, "worktree", "add", "--detach", staleWorktreePath, "main")
	if err != nil {
		t.Fatalf("git worktree add failed: %v", err)
	}
	_, err = executor.Shell(ctx, repoDir, fmt.Sprintf("rm -rf %s", staleWorktreePath))
	if err != nil {
		t.Fatalf("rm -rf stale worktree failed: %v", err)
	}

	// verify that stale ref exists (worktree list still shows it)
	out, err := executor.Git(ctx, repoDir, "worktree", "list")
	if err != nil {
		t.Fatalf("git worktree list failed: %v", err)
	}
	if !strings.Contains(string(out), "stale-wt") {
		t.Fatalf("expected stale worktree ref in list, got:\n%s", string(out))
	}

	pool := NewWorktreePool(executor, repoDir, "main", "", 1)

	// when — Init should succeed because prune cleans the stale ref
	err = pool.Init(ctx)

	// then
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
}

func TestWorktreePool_Init_RunsSetupCmd(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "touch .setup-done", 2)

	// when
	err := pool.Init(ctx)

	// then
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// verify marker file .setup-done exists in each worktree
	for i := range 2 {
		path := <-pool.workers
		_, err := executor.Shell(ctx, path, "test -f .setup-done")
		if err != nil {
			t.Errorf("expected .setup-done in worker %d %s, but file does not exist", i, path)
		}
		pool.workers <- path // put back
	}
}

func TestWorktreePool_Acquire_ReturnsValidPath(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 1)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// when
	path := pool.Acquire()

	// then
	if path == "" {
		t.Fatal("Acquire returned empty path")
	}
	if !strings.HasPrefix(path, repoDir) {
		t.Errorf("expected path under %s, got %s", repoDir, path)
	}

	// verify it is a valid git worktree
	out, err := executor.Git(ctx, path, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("rev-parse failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		t.Errorf("expected 'true', got %q", strings.TrimSpace(string(out)))
	}
}

func TestWorktreePool_Release_ResetsState(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 1)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	path := pool.Acquire()

	// dirty the worktree: create a file and a branch
	_, err := executor.Shell(ctx, path, "touch dirty-file.txt")
	if err != nil {
		t.Fatalf("touch failed: %v", err)
	}
	_, err = executor.Git(ctx, path, "checkout", "-b", "dirty-branch")
	if err != nil {
		t.Fatalf("checkout -b failed: %v", err)
	}

	// when
	err = pool.Release(ctx, path)

	// then
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// verify: status --porcelain is clean
	out, err := executor.Git(ctx, path, "status", "--porcelain")
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected clean status, got: %q", string(out))
	}

	// verify: HEAD is detached
	out, err = executor.Git(ctx, path, "symbolic-ref", "HEAD")
	if err == nil {
		t.Errorf("expected error (detached HEAD), but symbolic-ref succeeded with: %s", string(out))
	}
}

func TestWorktreePool_Release_ReturnsToPool(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 1)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// when
	path1 := pool.Acquire()
	err := pool.Release(ctx, path1)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}
	path2 := pool.Acquire()

	// then — reused the same path
	if path1 != path2 {
		t.Errorf("expected same path after release, got path1=%q path2=%q", path1, path2)
	}
}

func TestWorktreePool_Shutdown_RemovesAll(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 3)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// acquire all 3 and release all 3
	paths := make([]string, 3)
	for i := range 3 {
		paths[i] = pool.Acquire()
	}
	for _, p := range paths {
		if err := pool.Release(ctx, p); err != nil {
			t.Fatalf("Release failed: %v", err)
		}
	}

	// when
	err := pool.Shutdown(ctx)

	// then
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// verify: git worktree list shows only 1 line (main repo)
	out, err := executor.Git(ctx, repoDir, "worktree", "list")
	if err != nil {
		t.Fatalf("git worktree list failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 worktree entry (main only), got %d:\n%s", len(lines), string(out))
	}
}

// === Edge case tests (P0-P3) ===

func TestWorktreePool_Release_ResetsCommittedChanges(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	// record the baseBranch commit hash for later comparison
	baseBranchOut, err := executor.Git(ctx, repoDir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD failed: %v", err)
	}
	baseBranchCommit := strings.TrimSpace(string(baseBranchOut))

	pool := NewWorktreePool(executor, repoDir, "main", "", 1)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	path := pool.Acquire()

	// simulate the real expedition workflow: branch, create file, add, commit
	_, err = executor.Git(ctx, path, "checkout", "-b", "feature-branch")
	if err != nil {
		t.Fatalf("checkout -b failed: %v", err)
	}
	_, err = executor.Shell(ctx, path, "echo 'expedition work' > expedition-file.txt")
	if err != nil {
		t.Fatalf("creating file failed: %v", err)
	}
	_, err = executor.Git(ctx, path, "add", "expedition-file.txt")
	if err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	_, err = executor.Git(ctx, path, "commit", "-m", "expedition commit")
	if err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// verify the commit exists before release
	_, err = executor.Git(ctx, path, "log", "--oneline", "-1")
	if err != nil {
		t.Fatalf("git log failed: %v", err)
	}

	// when
	err = pool.Release(ctx, path)

	// then
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// verify: status --porcelain is clean
	out, err := executor.Git(ctx, path, "status", "--porcelain")
	if err != nil {
		t.Fatalf("git status failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected clean status, got: %q", string(out))
	}

	// verify: HEAD is detached at baseBranch commit
	out, err = executor.Git(ctx, path, "symbolic-ref", "HEAD")
	if err == nil {
		t.Errorf("expected error (detached HEAD), but symbolic-ref succeeded with: %s", string(out))
	}

	headOut, err := executor.Git(ctx, path, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("rev-parse HEAD after release failed: %v", err)
	}
	if strings.TrimSpace(string(headOut)) != baseBranchCommit {
		t.Errorf("expected HEAD at %s, got %s", baseBranchCommit, strings.TrimSpace(string(headOut)))
	}

	// verify: the committed file is gone
	_, err = executor.Shell(ctx, path, "test -f expedition-file.txt")
	if err == nil {
		t.Error("expected expedition-file.txt to be removed after release, but it still exists")
	}
}

func TestWorktreePool_Init_SetupCmdFailure_PartialInit(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	// pool with setupCmd that always fails
	failingPool := NewWorktreePool(executor, repoDir, "main", "exit 1", 3)

	// when
	err := failingPool.Init(ctx)

	// then — Init should return error
	if err == nil {
		t.Fatal("expected Init to fail with failing setupCmd, got nil")
	}

	// verify: no workers completed (channel is empty) because setupCmd fails on worker-001
	if len(failingPool.workers) != 0 {
		t.Errorf("expected 0 workers in channel, got %d", len(failingPool.workers))
	}

	// verify self-healing: remove the partial worktree directories (simulating cleanup),
	// then a SECOND pool with the same repoDir can Init successfully
	// because prune cleans up the stale refs whose directories no longer exist
	poolDir := repoDir + "/.expedition/.run/worktrees"
	_, err = executor.Shell(ctx, repoDir, fmt.Sprintf("rm -rf %s", poolDir))
	if err != nil {
		t.Fatalf("rm -rf poolDir failed: %v", err)
	}

	healthyPool := NewWorktreePool(executor, repoDir, "main", "", 3)
	err = healthyPool.Init(ctx)
	if err != nil {
		t.Fatalf("second Init (self-healing) failed: %v", err)
	}

	if len(healthyPool.workers) != 3 {
		t.Errorf("expected 3 workers after self-healing Init, got %d", len(healthyPool.workers))
	}
}

func TestWorktreePool_Shutdown_AcquiredWorkersAlsoCleaned(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 2)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// acquire 1 worker (don't release it)
	_ = pool.Acquire()

	// when — shutdown should remove ALL workers including acquired ones
	err := pool.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// then: git worktree list shows only 1 entry (main only, all workers cleaned)
	out, err := executor.Git(ctx, repoDir, "worktree", "list")
	if err != nil {
		t.Fatalf("git worktree list failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 worktree entry (main only), got %d:\n%s", len(lines), string(out))
	}
}

func TestWorktreePool_Release_ResetsStagedChanges(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 1)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	path := pool.Acquire()

	// stage a file without committing
	_, err := executor.Shell(ctx, path, "echo 'staged content' > staged-file.txt")
	if err != nil {
		t.Fatalf("creating file failed: %v", err)
	}
	_, err = executor.Git(ctx, path, "add", ".")
	if err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// verify the file is staged before release
	statusBefore, err := executor.Git(ctx, path, "status", "--porcelain")
	if err != nil {
		t.Fatalf("git status before release failed: %v", err)
	}
	if !strings.Contains(string(statusBefore), "staged-file.txt") {
		t.Fatalf("expected staged-file.txt in status, got: %q", string(statusBefore))
	}

	// when
	err = pool.Release(ctx, path)

	// then
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// verify: status --porcelain is clean (staged changes are gone)
	out, err := executor.Git(ctx, path, "status", "--porcelain")
	if err != nil {
		t.Fatalf("git status after release failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected clean status after release, got: %q", string(out))
	}
}

func TestWorktreePool_Init_ContextCancellation(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 3)

	// cancel the context before calling Init
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()

	// when
	err := pool.Init(cancelledCtx)

	// then — Init should return error due to cancelled context
	if err == nil {
		t.Fatal("expected Init to fail with cancelled context, got nil")
	}

	// the pool should NOT have all workers available
	if len(pool.workers) == 3 {
		t.Error("expected fewer than 3 workers with cancelled context, but got 3")
	}
}

func TestWorktreePool_Acquire_BlocksWhenExhausted(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 1)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// exhaust the pool
	firstPath := pool.Acquire()

	// when — start a goroutine that tries to acquire (should block)
	var acquiredPath atomic.Value
	done := make(chan struct{})
	go func() {
		path := pool.Acquire()
		acquiredPath.Store(path)
		close(done)
	}()

	// wait 200ms and verify the goroutine is still blocked
	time.Sleep(200 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("expected Acquire to block, but it returned immediately")
	default:
		// still blocked — expected
	}

	// release the first worker
	err := pool.Release(ctx, firstPath)
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// wait for the goroutine to complete
	select {
	case <-done:
		// goroutine returned — expected
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Acquire to unblock after Release")
	}

	// then — verify the path is valid
	got, ok := acquiredPath.Load().(string)
	if !ok || got == "" {
		t.Fatal("Acquire returned empty or invalid path after unblocking")
	}
	if got != firstPath {
		t.Errorf("expected same path %q, got %q", firstPath, got)
	}

	// verify it is a valid git worktree
	out, err := executor.Git(ctx, got, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("rev-parse failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		t.Errorf("expected 'true', got %q", strings.TrimSpace(string(out)))
	}
}

// TestWorktreePool_Init_SelfHeals_LeakedWorktrees verifies that Init
// succeeds even when a previous run leaked worktrees (acquired but not released
// before shutdown/crash). This is the regression test for the resource leak bug
// where early returns in Paintress.Run skipped Release, leaving orphaned worktrees
// that blocked the next Init's `git worktree add`.
func TestWorktreePool_Init_SelfHeals_LeakedWorktrees(t *testing.T) {
	// given — simulate a leaked worktree from a previous run
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	// Run 1: init pool, acquire worker, shutdown WITHOUT releasing
	pool1 := NewWorktreePool(executor, repoDir, "main", "", 1)
	if err := pool1.Init(ctx); err != nil {
		t.Fatalf("pool1 Init failed: %v", err)
	}
	_ = pool1.Acquire()     // acquire but DON'T release — simulates early return bug
	_ = pool1.Shutdown(ctx) // drains channel (empty), prunes (no stale), worker-001 dir stays

	// when — Run 2: new pool tries to Init on the same repo
	pool2 := NewWorktreePool(executor, repoDir, "main", "", 1)
	err := pool2.Init(ctx)

	// then — Init should succeed (self-healing removes stale worktree before re-creating)
	if err != nil {
		t.Fatalf("pool2 Init should self-heal leaked worktrees, got error: %v", err)
	}

	// verify: worker is available and functional
	path := pool2.Acquire()
	if path == "" {
		t.Fatal("Acquire returned empty path after self-healing Init")
	}
	out, err := executor.Git(ctx, path, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("rev-parse failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		t.Errorf("expected 'true', got %q", strings.TrimSpace(string(out)))
	}
}

// failOnCheckoutGitExecutor wraps a real executor but fails on checkout commands.
type failOnCheckoutGitExecutor struct {
	real      *localGitExecutor
	failCount atomic.Int32
}

func (f *failOnCheckoutGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	if len(args) >= 2 && args[0] == "checkout" && args[1] == "--detach" {
		f.failCount.Add(1)
		return nil, fmt.Errorf("simulated checkout failure")
	}
	return f.real.Git(ctx, dir, args...)
}

func (f *failOnCheckoutGitExecutor) Shell(ctx context.Context, dir string, command string) ([]byte, error) {
	return f.real.Shell(ctx, dir, command)
}

func TestWorktreePool_Release_CheckoutFailure_RecyclesSlot(t *testing.T) {
	// given: init with real executor, then swap to failing executor for Release
	ctx := context.Background()
	realExecutor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(realExecutor, repoDir, "main", "", 1)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	path := pool.Acquire()

	// swap executor to one that fails checkout
	failingExecutor := &failOnCheckoutGitExecutor{real: realExecutor}
	pool.git = failingExecutor

	// when: Release encounters checkout failure
	err := pool.Release(ctx, path)

	// then: Release should recover via forceRecycle (not lose the slot)
	if err != nil {
		t.Fatalf("Release should recover via recycle, got error: %v", err)
	}

	// swap back to real executor for verification
	pool.git = realExecutor

	// verify: the slot is back in the pool (can acquire again without deadlocking)
	path2 := pool.Acquire()
	if path2 == "" {
		t.Fatal("expected to acquire a worker after recycled Release")
	}

	// verify: the recycled worktree is functional
	out, err := realExecutor.Git(ctx, path2, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("rev-parse failed on recycled worktree: %v", err)
	}
	if strings.TrimSpace(string(out)) != "true" {
		t.Errorf("expected 'true', got %q", strings.TrimSpace(string(out)))
	}
}

func TestWorktreePool_Shutdown_CleansAcquiredWorkers(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 2)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// acquire 1 worker (don't release it -- simulates in-flight expedition)
	_ = pool.Acquire()

	// when
	err := pool.Shutdown(ctx)
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// then: ALL worktrees should be removed, including the acquired one
	out, err := executor.Git(ctx, repoDir, "worktree", "list")
	if err != nil {
		t.Fatalf("git worktree list failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 worktree entry (main only), got %d:\n%s", len(lines), string(out))
	}
}

// TestWorktreePool_Shutdown_CancelledContext_LeavesOrphans documents that
// Shutdown with a cancelled context cannot remove worktrees. This motivates
// the caller (Paintress.Run) to use context.Background() for deferred Shutdown.
func TestWorktreePool_Shutdown_CancelledContext_LeavesOrphans(t *testing.T) {
	// given
	ctx := context.Background()
	executor := &localGitExecutor{}
	repoDir := initGitRepoForWorktreeWithCommit(t)

	pool := NewWorktreePool(executor, repoDir, "main", "", 2)
	if err := pool.Init(ctx); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// when — cancel context, then call Shutdown
	cancelledCtx, cancel := context.WithCancel(ctx)
	cancel()
	err := pool.Shutdown(cancelledCtx)

	// then — Shutdown should fail (cannot run git commands with cancelled context)
	if err == nil {
		t.Error("expected Shutdown with cancelled context to return error, got nil")
	}

	// worktrees should still exist (orphaned)
	out, err := executor.Git(ctx, repoDir, "worktree", "list")
	if err != nil {
		t.Fatalf("git worktree list failed: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		t.Errorf("expected orphaned worktrees (>1 entries), got %d:\n%s", len(lines), string(out))
	}
}

// mockGitExecutor is a test double for port.GitExecutor (used by ValidateBaseBranch tests).
type mockGitExecutor struct {
	gitFn   func(ctx context.Context, dir string, args ...string) ([]byte, error)
	shellFn func(ctx context.Context, dir string, command string) ([]byte, error)
}

func (m *mockGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	if m.gitFn != nil {
		return m.gitFn(ctx, dir, args...)
	}
	return nil, nil
}

func (m *mockGitExecutor) Shell(ctx context.Context, dir string, command string) ([]byte, error) {
	if m.shellFn != nil {
		return m.shellFn(ctx, dir, command)
	}
	return nil, nil
}

func TestValidateBaseBranch_NonexistentBranch(t *testing.T) {
	// given
	git := &mockGitExecutor{
		gitFn: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("fatal: Needed a single revision")
		},
	}

	// when
	err := ValidateBaseBranch(context.Background(), git, "/tmp/repo", "mian")

	// then
	if err == nil {
		t.Fatal("expected error for nonexistent branch")
	}
	if !strings.Contains(err.Error(), "mian") {
		t.Errorf("error should mention branch name, got: %v", err)
	}
}

func TestValidateBaseBranch_ExistingBranch(t *testing.T) {
	// given
	git := &mockGitExecutor{
		gitFn: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			return []byte("abc123\n"), nil
		},
	}

	// when
	err := ValidateBaseBranch(context.Background(), git, "/tmp/repo", "main")

	// then
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
