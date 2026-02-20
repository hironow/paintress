package paintress

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GitExecutor abstracts git command execution for testability.
type GitExecutor interface {
	Git(ctx context.Context, dir string, args ...string) ([]byte, error)
	Shell(ctx context.Context, dir string, command string) ([]byte, error)
}

// localGitExecutor runs git commands on the host filesystem.
type localGitExecutor struct{}

func (e *localGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func (e *localGitExecutor) Shell(ctx context.Context, dir string, command string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// WorktreePool manages a pool of git worktrees for parallel expedition workers.
type WorktreePool struct {
	git        GitExecutor
	baseBranch string
	repoDir    string      // original repository
	poolDir    string      // .expedition/worktrees/
	setupCmd   string      // command to run after worktree creation
	workers    chan string // available worktree paths
	size       int
}

// NewWorktreePool creates a new WorktreePool with the given configuration.
func NewWorktreePool(git GitExecutor, repoDir, baseBranch, setupCmd string, size int) *WorktreePool {
	return &WorktreePool{
		git:        git,
		baseBranch: baseBranch,
		repoDir:    repoDir,
		poolDir:    filepath.Join(repoDir, ".expedition", ".run", "worktrees"),
		setupCmd:   setupCmd,
		workers:    make(chan string, size),
		size:       size,
	}
}

// Init prunes stale worktree references and creates fresh worktrees for each worker.
func (wp *WorktreePool) Init(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "worktree_pool.init",
		trace.WithAttributes(attribute.Int("pool.size", wp.size)),
	)
	defer span.End()

	if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "prune"); err != nil {
		return fmt.Errorf("worktree prune: %w", err)
	}

	for i := range wp.size {
		name := fmt.Sprintf("worker-%03d", i+1)
		path := filepath.Join(wp.poolDir, name)

		// Force-remove any leftover worktree from a previous run (self-healing).
		wp.git.Git(ctx, wp.repoDir, "worktree", "remove", "-f", path)

		if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "add", "--detach", path, wp.baseBranch); err != nil {
			return fmt.Errorf("worktree add %s: %w", name, err)
		}

		if wp.setupCmd != "" {
			if _, err := wp.git.Shell(ctx, path, wp.setupCmd); err != nil {
				return fmt.Errorf("setup cmd in %s: %w", name, err)
			}
		}

		wp.workers <- path
	}

	return nil
}

// Acquire returns the path to an available worktree. Blocks if none available.
func (wp *WorktreePool) Acquire() string {
	return <-wp.workers
}

// Release resets the worktree to a clean state and returns it to the pool.
func (wp *WorktreePool) Release(ctx context.Context, path string) error {
	if _, err := wp.git.Git(ctx, path, "checkout", "--detach", wp.baseBranch); err != nil {
		return fmt.Errorf("checkout %s: %w", wp.baseBranch, err)
	}
	if _, err := wp.git.Git(ctx, path, "reset", "--hard", wp.baseBranch); err != nil {
		return fmt.Errorf("reset: %w", err)
	}
	if _, err := wp.git.Git(ctx, path, "clean", "-fd"); err != nil {
		return fmt.Errorf("clean: %w", err)
	}
	wp.workers <- path
	return nil
}

// Shutdown removes all worktrees and cleans up the pool.
func (wp *WorktreePool) Shutdown(ctx context.Context) error {
	for {
		select {
		case path := <-wp.workers:
			if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "remove", "-f", path); err != nil {
				return fmt.Errorf("worktree remove %s: %w", path, err)
			}
		default:
			goto done
		}
	}
done:
	if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "prune"); err != nil {
		return fmt.Errorf("worktree prune: %w", err)
	}
	return nil
}
