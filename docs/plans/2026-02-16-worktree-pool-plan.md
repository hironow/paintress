# WorktreePool Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement a git worktree pool that provides isolated execution environments for Paintress expeditions.

**Architecture:** Independent `WorktreePool` struct with `GitExecutor` interface for testability. Pre-warm pool on startup, Acquire/Release per expedition, Shutdown on exit. testcontainers-go for all tests.

**Tech Stack:** Go 1.26, git worktree, testcontainers-go, alpine/git Docker image

**Design Doc:** `docs/plans/2026-02-16-worktree-pool-design.md`

---

### Task 1: Add testcontainers-go dependency

**Files:**
- Modify: `go.mod`

**Step 1: Add testcontainers-go**

Run: `go get github.com/testcontainers/testcontainers-go`

**Step 2: Verify**

Run: `go mod tidy`
Expected: `go.mod` and `go.sum` updated with testcontainers-go

**Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "feat: add testcontainers-go dependency"
```

---

### Task 2: GitExecutor interface + localGitExecutor

**Files:**
- Create: `worktree.go`
- Create: `worktree_test.go`

**Step 1: Write the failing test**

```go
// worktree_test.go
package main

import (
	"context"
	"os/exec"
	"testing"
)

func TestLocalGitExecutor_RunsGitCommand(t *testing.T) {
	// given
	dir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s", string(out))
	}

	executor := &localGitExecutor{}

	// when
	out, err := executor.Git(context.Background(), dir, "status")

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(out) == 0 {
		t.Error("expected output from git status")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestLocalGitExecutor_RunsGitCommand -v`
Expected: FAIL — `localGitExecutor` type not defined

**Step 3: Write minimal implementation**

```go
// worktree.go
package main

import (
	"context"
	"os/exec"
)

// GitExecutor abstracts git command execution for testability.
type GitExecutor interface {
	Git(ctx context.Context, dir string, args ...string) ([]byte, error)
}

// localGitExecutor runs git commands on the host filesystem.
type localGitExecutor struct{}

func (e *localGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestLocalGitExecutor_RunsGitCommand -v`
Expected: PASS

**Step 5: Commit**

```bash
git add worktree.go worktree_test.go
git commit -m "feat: add GitExecutor interface and localGitExecutor"
```

---

### Task 3: containerGitExecutor (testcontainers-go)

**Files:**
- Modify: `worktree_test.go`

**Step 1: Write the failing test**

```go
// worktree_test.go — add to existing file

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// containerGitExecutor runs git inside a Docker container via testcontainers-go.
type containerGitExecutor struct {
	ctr testcontainers.Container
}

func (e *containerGitExecutor) Git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	fullArgs := append([]string{"git", "-C", dir}, args...)
	exitCode, reader, err := e.ctr.Exec(ctx, fullArgs)
	if err != nil {
		return nil, fmt.Errorf("exec failed: %w", err)
	}
	out, _ := io.ReadAll(reader)
	if exitCode != 0 {
		return out, fmt.Errorf("git exited %d: %s", exitCode, string(out))
	}
	return out, nil
}

// setupGitContainer starts an alpine/git container and returns a GitExecutor.
func setupGitContainer(t *testing.T) GitExecutor {
	t.Helper()
	ctx := context.Background()
	ctr, err := testcontainers.Run(ctx, "alpine/git:latest",
		testcontainers.WithCmd([]string{"sleep", "infinity"}),
		testcontainers.WithWaitStrategy(wait.ForExec([]string{"git", "--version"})),
	)
	if err != nil {
		t.Fatalf("start container: %v", err)
	}
	testcontainers.CleanupContainer(t, ctr)

	// Configure git inside container
	for _, args := range [][]string{
		{"git", "config", "--global", "user.email", "test@test.com"},
		{"git", "config", "--global", "user.name", "test"},
		{"git", "config", "--global", "init.defaultBranch", "main"},
	} {
		if code, _, err := ctr.Exec(ctx, args); err != nil || code != 0 {
			t.Fatalf("git config failed: %v (exit %d)", err, code)
		}
	}

	return &containerGitExecutor{ctr: ctr}
}

func TestContainerGitExecutor_RunsGitCommand(t *testing.T) {
	// given
	git := setupGitContainer(t)
	ctx := context.Background()

	// Init a repo inside the container
	_, err := git.Git(ctx, "/tmp", "init", "/tmp/test-repo")
	if err != nil {
		t.Fatalf("git init: %v", err)
	}

	// when
	out, err := git.Git(ctx, "/tmp/test-repo", "status")

	// then
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(string(out), "On branch main") {
		t.Errorf("expected 'On branch main' in output, got: %s", string(out))
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./... -run TestContainerGitExecutor -v -timeout 120s`
Expected: PASS (container starts, git runs inside it)

Note: First run may be slow (pulling alpine/git image). Subsequent runs use cache.

**Step 3: Commit**

```bash
git add worktree_test.go
git commit -m "feat: add containerGitExecutor for testcontainers-go"
```

---

### Task 4: WorktreePool struct + NewWorktreePool + Init

**Files:**
- Modify: `worktree.go`
- Modify: `worktree_test.go`

**Step 1: Write the failing test for Init**

```go
func TestWorktreePool_Init_CreatesWorktrees(t *testing.T) {
	// given
	git := setupGitContainer(t)
	ctx := context.Background()
	repoDir := "/tmp/repo"

	// Create a bare repo with a commit
	git.Git(ctx, "/tmp", "init", repoDir)
	git.Git(ctx, repoDir, "commit", "--allow-empty", "-m", "init")

	poolDir := repoDir + "/.expedition/worktrees"
	pool := NewWorktreePool(git, repoDir, "main", "", 3)

	// when
	err := pool.Init(ctx)

	// then
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify 3 worktrees were created
	out, err := git.Git(ctx, repoDir, "worktree", "list")
	if err != nil {
		t.Fatalf("worktree list: %v", err)
	}
	// Main repo + 3 worktrees = 4 lines
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 4 {
		t.Errorf("expected 4 worktree entries (1 main + 3 workers), got %d: %s", len(lines), string(out))
	}

	// Verify all 3 workers are in the channel
	for i := 0; i < 3; i++ {
		select {
		case path := <-pool.workers:
			if !strings.Contains(path, "worker-") {
				t.Errorf("expected worker path, got: %s", path)
			}
			pool.workers <- path // put back
		default:
			t.Errorf("expected worker %d in pool, channel empty", i)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestWorktreePool_Init_CreatesWorktrees -v -timeout 120s`
Expected: FAIL — `NewWorktreePool` not defined

**Step 3: Write minimal implementation**

```go
// worktree.go — add to existing file

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

// WorktreePool manages a pool of git worktrees for isolated expedition execution.
type WorktreePool struct {
	git        GitExecutor
	baseBranch string
	repoDir    string
	poolDir    string
	setupCmd   string
	workers    chan string
	size       int
}

// NewWorktreePool creates a new pool. Call Init() to pre-warm worktrees.
func NewWorktreePool(git GitExecutor, repoDir, baseBranch, setupCmd string, size int) *WorktreePool {
	return &WorktreePool{
		git:        git,
		baseBranch: baseBranch,
		repoDir:    repoDir,
		poolDir:    filepath.Join(repoDir, ".expedition", "worktrees"),
		setupCmd:   setupCmd,
		workers:    make(chan string, size),
		size:       size,
	}
}

// Init pre-warms the pool by creating worktrees and optionally running setup commands.
func (wp *WorktreePool) Init(ctx context.Context) error {
	// Prune stale worktree references from previous crashes
	wp.git.Git(ctx, wp.repoDir, "worktree", "prune")

	for i := 0; i < wp.size; i++ {
		name := fmt.Sprintf("worker-%03d", i)
		path := filepath.Join(wp.poolDir, name)

		if _, err := wp.git.Git(ctx, wp.repoDir, "worktree", "add", path, wp.baseBranch); err != nil {
			return fmt.Errorf("create worktree %s: %w", name, err)
		}

		if wp.setupCmd != "" {
			if _, err := wp.git.Git(ctx, path, "-c", "!", "sh", "-c", wp.setupCmd); err != nil {
				// setupCmd is not a git command — need shell execution
				// This will be handled differently; for now skip in Git interface
			}
		}

		wp.workers <- path
	}
	return nil
}
```

Wait — `setupCmd` is a shell command, not a git command. The `GitExecutor` interface only supports git. We need a separate `ShellExecutor` or handle it at a higher level.

**Design decision**: `setupCmd` runs via `exec.Command("sh", "-c", setupCmd)` directly (not through GitExecutor). In container tests, we execute it via `ctr.Exec()`. Add a `ShellExec` method to the executor interface or keep it separate.

Simplest approach: extend `GitExecutor` to a more general `CommandExecutor`:

```go
type CommandExecutor interface {
	Git(ctx context.Context, dir string, args ...string) ([]byte, error)
	Shell(ctx context.Context, dir string, command string) ([]byte, error)
}
```

Update implementation in this step.

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestWorktreePool_Init_CreatesWorktrees -v -timeout 120s`
Expected: PASS

**Step 5: Commit**

```bash
git add worktree.go worktree_test.go
git commit -m "feat: add WorktreePool with Init pre-warming"
```

---

### Task 5: Init — prunes stale worktrees

**Files:**
- Modify: `worktree_test.go`

**Step 1: Write the failing test**

```go
func TestWorktreePool_Init_PrunesStale(t *testing.T) {
	// given — create a worktree, then manually remove its directory
	// (simulates a crash that left a stale reference)
	git := setupGitContainer(t)
	ctx := context.Background()
	repoDir := "/tmp/repo-prune"

	git.Git(ctx, "/tmp", "init", repoDir)
	git.Git(ctx, repoDir, "commit", "--allow-empty", "-m", "init")

	// Create a worktree, then remove its directory (stale reference)
	stalePath := repoDir + "/.expedition/worktrees/stale-worker"
	git.Git(ctx, repoDir, "worktree", "add", stalePath, "main")
	// Remove the directory but leave the git reference
	git.Git(ctx, stalePath, "rm", "-rf", stalePath) // or use Shell

	pool := NewWorktreePool(git, repoDir, "main", "", 1)

	// when
	err := pool.Init(ctx)

	// then — should succeed (prune cleaned the stale ref)
	if err != nil {
		t.Fatalf("Init should succeed after pruning stale worktrees: %v", err)
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./... -run TestWorktreePool_Init_PrunesStale -v -timeout 120s`
Expected: PASS (prune already in Init)

**Step 3: Commit**

```bash
git add worktree_test.go
git commit -m "test: verify Init prunes stale worktrees"
```

---

### Task 6: Init — runs setupCmd

**Files:**
- Modify: `worktree_test.go`

**Step 1: Write the failing test**

```go
func TestWorktreePool_Init_RunsSetupCmd(t *testing.T) {
	// given
	git := setupGitContainer(t)
	ctx := context.Background()
	repoDir := "/tmp/repo-setup"

	git.Git(ctx, "/tmp", "init", repoDir)
	git.Git(ctx, repoDir, "commit", "--allow-empty", "-m", "init")

	// setupCmd creates a marker file
	pool := NewWorktreePool(git, repoDir, "main", "touch .setup-done", 2)

	// when
	err := pool.Init(ctx)

	// then
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Verify marker file exists in each worktree
	for i := 0; i < 2; i++ {
		path := <-pool.workers
		markerPath := path + "/.setup-done"
		// Check marker exists via shell
		out, err := git.Shell(ctx, path, "ls .setup-done")
		if err != nil {
			t.Errorf("setup-cmd did not run in %s: %v", path, err)
		}
		pool.workers <- path
	}
}
```

**Step 2: Run test, implement Shell method if needed, verify pass**

**Step 3: Commit**

```bash
git add worktree.go worktree_test.go
git commit -m "feat: WorktreePool.Init runs setupCmd in each worktree"
```

---

### Task 7: Acquire + Release

**Files:**
- Modify: `worktree.go`
- Modify: `worktree_test.go`

**Step 1: Write the failing test for Acquire**

```go
func TestWorktreePool_Acquire_ReturnsValidPath(t *testing.T) {
	// given
	git := setupGitContainer(t)
	ctx := context.Background()
	repoDir := "/tmp/repo-acquire"

	git.Git(ctx, "/tmp", "init", repoDir)
	git.Git(ctx, repoDir, "commit", "--allow-empty", "-m", "init")

	pool := NewWorktreePool(git, repoDir, "main", "", 2)
	pool.Init(ctx)

	// when
	path := pool.Acquire()

	// then
	if path == "" {
		t.Fatal("Acquire returned empty path")
	}
	if !strings.HasPrefix(path, repoDir) {
		t.Errorf("expected path under repo, got: %s", path)
	}

	// Verify the directory is a valid git worktree
	out, err := git.Git(ctx, path, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		t.Fatalf("not a valid worktree: %v", err)
	}
	if !strings.Contains(string(out), "true") {
		t.Errorf("expected 'true', got: %s", string(out))
	}
}
```

**Step 2: Write the failing test for Release**

```go
func TestWorktreePool_Release_ResetsState(t *testing.T) {
	// given
	git := setupGitContainer(t)
	ctx := context.Background()
	repoDir := "/tmp/repo-release"

	git.Git(ctx, "/tmp", "init", repoDir)
	git.Git(ctx, repoDir, "commit", "--allow-empty", "-m", "init")

	pool := NewWorktreePool(git, repoDir, "main", "", 1)
	pool.Init(ctx)

	path := pool.Acquire()

	// Dirty the worktree (simulate expedition work)
	git.Shell(ctx, path, "echo 'dirty' > dirty.txt")
	git.Git(ctx, path, "checkout", "-b", "feat/test")

	// when
	err := pool.Release(ctx, path)

	// then
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	// Verify clean state: on baseBranch, no dirty files
	out, err := git.Git(ctx, path, "branch", "--show-current")
	if err != nil {
		t.Fatalf("branch check: %v", err)
	}
	if !strings.Contains(string(out), "main") {
		t.Errorf("expected branch 'main', got: %s", string(out))
	}

	out, err = git.Git(ctx, path, "status", "--porcelain")
	if err != nil {
		t.Fatalf("status check: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("expected clean worktree, got: %s", string(out))
	}
}

func TestWorktreePool_Release_ReturnsToPool(t *testing.T) {
	// given
	git := setupGitContainer(t)
	ctx := context.Background()
	repoDir := "/tmp/repo-reuse"

	git.Git(ctx, "/tmp", "init", repoDir)
	git.Git(ctx, repoDir, "commit", "--allow-empty", "-m", "init")

	pool := NewWorktreePool(git, repoDir, "main", "", 1)
	pool.Init(ctx)

	path1 := pool.Acquire()
	pool.Release(ctx, path1)

	// when — acquire again
	path2 := pool.Acquire()

	// then — should get the same path back
	if path1 != path2 {
		t.Errorf("expected reused path %s, got %s", path1, path2)
	}
}
```

**Step 3: Write minimal implementation**

```go
// Acquire returns the path to an available worktree. Blocks if none available.
func (wp *WorktreePool) Acquire() string {
	return <-wp.workers
}

// Release resets the worktree to a clean state and returns it to the pool.
func (wp *WorktreePool) Release(ctx context.Context, path string) error {
	if _, err := wp.git.Git(ctx, path, "checkout", wp.baseBranch); err != nil {
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
```

**Step 4: Run tests to verify they pass**

Run: `go test ./... -run "TestWorktreePool_(Acquire|Release)" -v -timeout 120s`
Expected: PASS

**Step 5: Commit**

```bash
git add worktree.go worktree_test.go
git commit -m "feat: add WorktreePool.Acquire and Release"
```

---

### Task 8: Shutdown

**Files:**
- Modify: `worktree.go`
- Modify: `worktree_test.go`

**Step 1: Write the failing test**

```go
func TestWorktreePool_Shutdown_RemovesAll(t *testing.T) {
	// given
	git := setupGitContainer(t)
	ctx := context.Background()
	repoDir := "/tmp/repo-shutdown"

	git.Git(ctx, "/tmp", "init", repoDir)
	git.Git(ctx, repoDir, "commit", "--allow-empty", "-m", "init")

	pool := NewWorktreePool(git, repoDir, "main", "", 3)
	pool.Init(ctx)

	// Drain the pool so all workers are returned for shutdown
	paths := make([]string, 3)
	for i := 0; i < 3; i++ {
		paths[i] = pool.Acquire()
	}
	for _, p := range paths {
		pool.Release(ctx, p)
	}

	// when
	err := pool.Shutdown(ctx)

	// then
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Verify only main repo remains in worktree list
	out, _ := git.Git(ctx, repoDir, "worktree", "list")
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected only main repo in worktree list, got %d entries: %s", len(lines), string(out))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./... -run TestWorktreePool_Shutdown -v -timeout 120s`
Expected: FAIL — `Shutdown` not defined

**Step 3: Write minimal implementation**

```go
// Shutdown removes all worktrees and cleans up the pool directory.
func (wp *WorktreePool) Shutdown(ctx context.Context) error {
	// Drain all workers from the channel
	for {
		select {
		case path := <-wp.workers:
			wp.git.Git(ctx, wp.repoDir, "worktree", "remove", "-f", path)
		default:
			goto done
		}
	}
done:
	wp.git.Git(ctx, wp.repoDir, "worktree", "prune")
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./... -run TestWorktreePool_Shutdown -v -timeout 120s`
Expected: PASS

**Step 5: Commit**

```bash
git add worktree.go worktree_test.go
git commit -m "feat: add WorktreePool.Shutdown"
```

---

### Task 9: CLI flags (--workers, --setup-cmd)

**Files:**
- Modify: `main.go:18-30` (Config struct)
- Modify: `main.go:56-122` (parseFlags)

**Step 1: Add fields to Config**

```go
type Config struct {
	// ...existing fields...
	Workers   int    // Number of worktrees in pool (0 = no worktree)
	SetupCmd  string // Command to run after worktree creation
}
```

**Step 2: Add flags to parseFlags**

```go
flag.IntVar(&cfg.Workers, "workers", 1, "Number of worktrees in pool (0 = direct execution)")
flag.StringVar(&cfg.SetupCmd, "setup-cmd", "", "Command to run after worktree creation (e.g. 'bun install')")
```

**Step 3: Run existing tests to verify no regression**

Run: `go test ./... -count=1 -timeout 120s`
Expected: PASS

**Step 4: Commit**

```bash
git add main.go
git commit -m "feat: add --workers and --setup-cmd CLI flags"
```

---

### Task 10: Expedition.WorkDir field

**Files:**
- Modify: `expedition.go:38-51` (Expedition struct)
- Modify: `expedition.go:116` (cmd.Dir)

**Step 1: Add WorkDir field**

```go
type Expedition struct {
	Number    int
	Continent string
	WorkDir   string // execution directory (worktree path or Continent)
	Config    Config
	// ...rest unchanged...
}
```

**Step 2: Use WorkDir for cmd.Dir with fallback**

Change `expedition.go:116`:
```go
// Before:
cmd.Dir = e.Continent

// After:
workDir := e.WorkDir
if workDir == "" {
	workDir = e.Continent
}
cmd.Dir = workDir
```

**Step 3: Run existing tests to verify no regression**

Run: `go test ./... -count=1 -timeout 120s`
Expected: PASS (WorkDir is empty everywhere, falls back to Continent)

**Step 4: Commit**

```bash
git add expedition.go
git commit -m "feat: add Expedition.WorkDir with Continent fallback"
```

---

### Task 11: Integrate WorktreePool into Paintress.Run

**Files:**
- Modify: `paintress.go:16-27` (Paintress struct)
- Modify: `paintress.go:29-60` (NewPaintress)
- Modify: `paintress.go:62-257` (Run method)

**Step 1: Add pool field and initialization**

```go
type Paintress struct {
	// ...existing fields...
	pool *WorktreePool // nil when --workers=0
}
```

In `Run()`, before the expedition loop:

```go
// Initialize worktree pool if workers > 0
if p.config.Workers > 0 {
	p.pool = NewWorktreePool(
		&localGitExecutor{},
		p.config.Continent,
		p.config.BaseBranch,
		p.config.SetupCmd,
		p.config.Workers,
	)
	if err := p.pool.Init(ctx); err != nil {
		LogError("%s", fmt.Sprintf("worktree pool init failed: %v", err))
		return 1
	}
	defer p.pool.Shutdown(ctx)
}
```

**Step 2: Acquire/Release around expedition**

In the expedition loop, wrap the expedition execution:

```go
// Before expedition creation (around line 134)
var workDir string
if p.pool != nil {
	workDir = p.pool.Acquire()
	defer func() {
		if workDir != "" {
			p.pool.Release(ctx, workDir)
		}
	}()
}

expedition := &Expedition{
	Number:    exp,
	Continent: p.config.Continent,
	WorkDir:   workDir,
	// ...rest unchanged...
}
```

Note: The `defer` inside the for loop accumulates — use explicit release instead:

```go
var workDir string
if p.pool != nil {
	workDir = p.pool.Acquire()
}

expedition := &Expedition{
	Number:    exp,
	Continent: p.config.Continent,
	WorkDir:   workDir,
	// ...rest unchanged...
}

// ... expedition execution ...

// Release worktree at end of loop iteration (after review loop, after return-to-base)
if p.pool != nil && workDir != "" {
	if err := p.pool.Release(ctx, workDir); err != nil {
		LogWarn("worktree release: %v", err)
	}
}
```

**Step 3: Update review loop to use workDir**

Change `paintress.go:296` and `paintress.go:330` and `paintress.go:370`:

```go
// Before:
RunReview(reviewCtx, p.config.ReviewCmd, p.config.Continent)
gitCmd.Dir = p.config.Continent
cmd.Dir = p.config.Continent

// After — use workDir with fallback:
reviewDir := workDir
if reviewDir == "" {
	reviewDir = p.config.Continent
}
RunReview(reviewCtx, p.config.ReviewCmd, reviewDir)
gitCmd.Dir = reviewDir
cmd.Dir = reviewDir
```

This requires passing `workDir` to `runReviewLoop`. Update signature:

```go
func (p *Paintress) runReviewLoop(ctx context.Context, report *ExpeditionReport, budget time.Duration, workDir string)
```

**Step 4: Update return-to-base to use workDir**

Change `paintress.go:240-242`:

```go
// Before:
gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
gitCmd.Dir = p.config.Continent

// After (skip if worktree — Release handles this):
if p.pool == nil {
	gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
	gitCmd.Dir = p.config.Continent
	_ = gitCmd.Run()
}
```

**Step 5: Run existing tests**

Run: `go test ./... -count=1 -timeout 120s`
Expected: PASS

**Step 6: Commit**

```bash
git add paintress.go
git commit -m "feat: integrate WorktreePool into Paintress.Run"
```

---

### Task 12: Update .gitignore for worktrees directory

**Files:**
- Modify: `.expedition/.gitignore` (via `validateContinent` or manual)

**Step 1: Update validateContinent**

```go
// In main.go, update the gitignore content:
os.WriteFile(gitignore, []byte(".logs/\nworktrees/\n"), 0644)
```

**Step 2: Run tests**

Run: `go test ./... -count=1 -timeout 120s`
Expected: PASS

**Step 3: Commit**

```bash
git add main.go
git commit -m "feat: gitignore .expedition/worktrees/"
```

---

### Task 13: Full integration test — run all tests

**Step 1: Run all tests**

Run: `go test ./... -count=1 -timeout 180s -v`
Expected: ALL PASS

**Step 2: Verify build**

Run: `go build -o /dev/null .`
Expected: No errors

**Step 3: Final commit if any remaining changes**

```bash
git add -A
git commit -m "test: verify full integration of WorktreePool"
```
