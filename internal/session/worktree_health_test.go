package session

// white-box-reason: session internals: tests AcquireContext and worktree health check behavior

import (
	"context"
	"testing"
)

// TestAcquireContext_HealthyWorktreeReturnsPath verifies AcquireContext returns
// a valid worktree path when the worktree is healthy.
func TestAcquireContext_HealthyWorktreeReturnsPath(t *testing.T) {
	// given: a real worktree pool with one healthy worktree
	dir := initGitRepoForWorktreeWithCommit(t)
	git := &localGitExecutor{}
	pool := NewWorktreePool(git, dir, "main", "", 1)
	if err := pool.Init(context.Background()); err != nil {
		t.Fatalf("pool.Init: %v", err)
	}
	defer pool.Shutdown(context.Background())

	// when
	path, err := pool.AcquireContext(context.Background())

	// then
	if err != nil {
		t.Fatalf("AcquireContext returned error: %v", err)
	}
	if path == "" {
		t.Error("AcquireContext returned empty path")
	}

	// cleanup: release back to pool
	pool.Release(context.Background(), path)
}

// TestAcquireContext_CanceledContextReturnsError verifies AcquireContext
// returns an error when the context is canceled before a worktree is available.
func TestAcquireContext_CanceledContextReturnsError(t *testing.T) {
	// given: pool with no available worktrees (all acquired)
	dir := initGitRepoForWorktreeWithCommit(t)
	git := &localGitExecutor{}
	pool := NewWorktreePool(git, dir, "main", "", 1)
	if err := pool.Init(context.Background()); err != nil {
		t.Fatalf("pool.Init: %v", err)
	}
	defer pool.Shutdown(context.Background())

	// Acquire the only available slot to drain the pool
	held, err := pool.AcquireContext(context.Background())
	if err != nil {
		t.Fatalf("first AcquireContext: %v", err)
	}
	defer pool.Release(context.Background(), held)

	// Create a pre-canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// when: try to acquire with a canceled context (pool is empty)
	_, err = pool.AcquireContext(ctx)

	// then: must return an error due to context cancellation
	if err == nil {
		t.Error("AcquireContext should return error when context is canceled")
	}
}

// TestAcquire_LegacyWrapperStillWorks verifies the legacy Acquire() method
// still functions as a convenience wrapper around AcquireContext.
func TestAcquire_LegacyWrapperStillWorks(t *testing.T) {
	// given
	dir := initGitRepoForWorktreeWithCommit(t)
	git := &localGitExecutor{}
	pool := NewWorktreePool(git, dir, "main", "", 1)
	if err := pool.Init(context.Background()); err != nil {
		t.Fatalf("pool.Init: %v", err)
	}
	defer pool.Shutdown(context.Background())

	// when: use legacy Acquire() method
	path := pool.Acquire()

	// then
	if path == "" {
		t.Error("legacy Acquire() returned empty path")
	}

	// cleanup
	pool.Release(context.Background(), path)
}
