package session

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Compile-time check that localGitExecutor implements port.GitExecutor.
var _ port.GitExecutor = (*localGitExecutor)(nil)

// localGitExecutor runs git commands on the host filesystem.
type localGitExecutor struct{}

func (e *localGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func (e *localGitExecutor) Shell(ctx context.Context, dir string, command string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, shellName(), shellFlag(), command) // nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command,semgrep.lod-excessive-dot-chain -- command is from validated Config.SetupCmd, not user input; exec.CommandContext form is standard os/exec API shape [permanent]
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

// WorktreePool manages a pool of git worktrees for parallel expedition workers.
type WorktreePool struct {
	git        port.GitExecutor
	baseBranch string
	repoDir    string      // original repository
	poolDir    string      // .expedition/worktrees/
	setupCmd   string      // command to run after worktree creation
	workers    chan string // available worktree paths
	size       int
	allPaths   []string // all created worktree paths (for complete shutdown cleanup)
}

// NewWorktreePool creates a new WorktreePool with the given configuration.
func NewWorktreePool(git port.GitExecutor, repoDir, baseBranch, setupCmd string, size int) *WorktreePool { // nosemgrep: domain-primitives.multiple-string-params-go -- repoDir/baseBranch/setupCmd are semantically distinct config params [permanent]
	return &WorktreePool{
		git:        git,
		baseBranch: baseBranch,
		repoDir:    repoDir,
		poolDir:    filepath.Join(repoDir, domain.StateDir, ".run", "worktrees"),
		setupCmd:   setupCmd,
		workers:    make(chan string, size),
		size:       size,
	}
}

// ParseBaseBranch verifies that the given branch exists as a local git ref,
// returning the branch name or an error.
func ParseBaseBranch(ctx context.Context, git port.GitExecutor, repoDir, branch string) (string, error) {
	_, err := git.Git(ctx, repoDir, "rev-parse", "--verify", branch)
	if err != nil {
		return "", fmt.Errorf("base branch %q does not exist as a git ref: %w", branch, err)
	}
	return branch, nil
}

// ValidateBaseBranch verifies that the given branch exists as a local git ref.
//
// Deprecated: prefer ParseBaseBranch which returns the validated branch name.
func ValidateBaseBranch(ctx context.Context, git port.GitExecutor, repoDir, branch string) error { // nosemgrep: parse-dont-validate.validate-returns-error-only-go -- backward-compat wrapper; ParseBaseBranch is the canonical parse function [permanent]
	_, err := ParseBaseBranch(ctx, git, repoDir, branch)
	return err
}

// Init prunes stale worktree references and creates fresh worktrees for each worker.
func (wp *WorktreePool) Init(ctx context.Context) error {
	ctx, span := platform.Tracer.Start(ctx, "worktree_pool.init",
		trace.WithAttributes(attribute.Int("pool.size", wp.size)),
	)
	defer span.End()

	if err := ValidateBaseBranch(ctx, wp.git, wp.repoDir, wp.baseBranch); err != nil {
		return err
	}

	if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "prune"); err != nil {
		return fmt.Errorf("worktree prune: %w", err)
	}

	for i := range wp.size {
		name := fmt.Sprintf("worker-%03d", i+1)
		path := filepath.Join(wp.poolDir, name)

		wp.git.Git(ctx, wp.repoDir, "worktree", "remove", "-f", path)

		if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "add", "--detach", path, wp.baseBranch); err != nil {
			return fmt.Errorf("worktree add %s: %w", name, err)
		}

		if wp.setupCmd != "" {
			if _, err := wp.git.Shell(ctx, path, wp.setupCmd); err != nil {
				return fmt.Errorf("setup cmd in %s: %w", name, err)
			}
		}

		wp.allPaths = append(wp.allPaths, path)
		wp.workers <- path
	}

	return nil
}

// AcquireContext returns the path to an available worktree, respecting context cancellation.
// Runs a git status health check on the acquired worktree. If the worktree is
// unhealthy, it is automatically recycled and a fresh one is returned.
// Returns an error if the context is canceled before a worktree becomes available.
func (wp *WorktreePool) AcquireContext(ctx context.Context) (string, error) {
	select {
	case path := <-wp.workers:
		// Health check: verify the worktree is accessible via git status.
		if _, err := wp.git.Git(ctx, path, "status", "--short"); err != nil {
			// Worktree is unhealthy — attempt recreation.
			if recycleErr := wp.forceRecycle(ctx, path); recycleErr != nil {
				return "", fmt.Errorf("worktree health check failed and recycle failed: %w", recycleErr)
			}
			// Pick the freshly recycled worktree.
			select {
			case path = <-wp.workers:
			case <-ctx.Done():
				return "", ctx.Err()
			}
		}
		return path, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// Acquire returns the path to an available worktree. Blocks if none available.
// Deprecated: prefer AcquireContext for context-aware acquisition.
func (wp *WorktreePool) Acquire() string {
	return <-wp.workers
}

// Release resets the worktree to a clean state and returns it to the pool.
// On checkout, reset, or clean failure, it force-recycles the worktree (remove + re-add)
// to prevent permanent pool slot loss. If forceRecycle itself fails, the path is
// still returned to the pool; the next Acquire's health check will re-attempt recycling.
func (wp *WorktreePool) Release(ctx context.Context, path string) error {
	if _, err := wp.git.Git(ctx, path, "checkout", "--detach", wp.baseBranch); err != nil {
		return wp.forceRecycle(ctx, path)
	}
	if _, err := wp.git.Git(ctx, path, "reset", "--hard", wp.baseBranch); err != nil {
		return wp.forceRecycle(ctx, path)
	}
	// clean failure means the worktree may have leftover state — recycle it
	// to avoid returning a dirty worktree to the pool.
	if _, err := wp.git.Git(ctx, path, "clean", "-fd", "-e", domain.StateDir); err != nil {
		return wp.forceRecycle(ctx, path)
	}
	wp.workers <- path
	return nil
}

// forceRecycle removes a corrupted worktree and re-creates it from scratch.
// This prevents permanent pool slot loss when checkout/reset fails.
func (wp *WorktreePool) forceRecycle(ctx context.Context, path string) error {
	if _, removeErr := wp.git.Git(ctx, wp.repoDir, "worktree", "remove", "-f", path); removeErr != nil {
		// Log but continue — the directory may already be gone, and --force add below can cope.
		_ = removeErr
	}

	if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "add", "--force", "--detach", path, wp.baseBranch); err != nil {
		// Return the path anyway to avoid permanent slot loss; Acquire's
		// health check will attempt recycling again on next use.
		wp.workers <- path
		return fmt.Errorf("forceRecycle worktree add %s: %w", path, err)
	}

	if wp.setupCmd != "" {
		if _, err := wp.git.Shell(ctx, path, wp.setupCmd); err != nil {
			wp.workers <- path
			return fmt.Errorf("forceRecycle setup cmd in %s: %w", path, err)
		}
	}

	wp.workers <- path
	return nil
}

// Shutdown removes all worktrees and cleans up the pool.
// It drains the channel to unblock any pending Acquires, then removes all
// worktrees tracked in allPaths (including those currently acquired by
// workers), preventing resource leaks on shutdown.
func (wp *WorktreePool) Shutdown(ctx context.Context) error {
	// Drain channel to unblock any pending Acquires.
drain:
	for {
		select {
		case <-wp.workers:
		default:
			break drain
		}
	}

	// Remove all worktrees (both acquired and released).
	var errs []error
	for _, path := range wp.allPaths {
		if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "remove", "-f", path); err != nil {
			errs = append(errs, fmt.Errorf("worktree remove %s: %w", path, err))
		}
	}

	if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "prune"); err != nil {
		errs = append(errs, fmt.Errorf("worktree prune: %w", err))
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
