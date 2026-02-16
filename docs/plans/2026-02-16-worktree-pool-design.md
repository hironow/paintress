# WorktreePool Design

**Date:** 2026-02-16
**Status:** Approved

## Purpose

Enable git worktree-based isolated execution for Paintress expeditions. This is the foundation for future parallel (Swarm Mode) execution, but the initial scope is the pool infrastructure only — the existing sequential loop will use worktrees one at a time.

## Architecture

### WorktreePool (Independent Struct)

```
+-------------------+       +-------------------+
|   Paintress.Run   | ----> | WorktreePool      |
|   (orchestrator)  |       |  Init()           |
+-------------------+       |  Acquire() -> path|
         |                  |  Release(path)    |
         v                  |  Shutdown()       |
+-------------------+       +-------------------+
|   Expedition.Run  |              |
|   cmd.Dir = path  |       +------+------+
+-------------------+       | worker-001/ |
                            | worker-002/ |
                            +-------------+
```

### GitExecutor Interface

Abstracts git command execution for testability with testcontainers-go.

```go
type GitExecutor interface {
    Git(ctx context.Context, dir string, args ...string) ([]byte, error)
}
```

- **Production**: `localGitExecutor` — runs `exec.Command("git", ...)` on host
- **Test**: `containerGitExecutor` — runs git inside Docker via `ctr.Exec()`

### WorktreePool Struct

```go
type WorktreePool struct {
    git        GitExecutor
    baseBranch string
    repoDir    string       // original repository (Continent)
    poolDir    string       // .expedition/worktrees/
    setupCmd   string       // command to run after worktree creation
    workers    chan string   // available worktree paths
    size       int
}
```

### Public API

| Method | Description |
|--------|-------------|
| `NewWorktreePool(git, repoDir, baseBranch, setupCmd, size)` | Create pool (does not allocate worktrees) |
| `Init(ctx) error` | Pre-warm: prune stale worktrees, create `size` worktrees, run setupCmd |
| `Acquire() string` | Get available worktree path (blocks if none available) |
| `Release(path) error` | Reset worktree state and return to pool |
| `Shutdown() error` | Remove all worktrees, prune, clean up directory |

## Lifecycle

### Init (Pre-warm)

1. `git worktree prune` — clean stale references from previous crashes
2. `mkdir -p .expedition/worktrees/`
3. For each worker (0..size-1):
   - `git worktree add .expedition/worktrees/worker-00N <baseBranch>`
   - `sh -c <setupCmd>` (if configured)
   - Push path to `workers` channel

### Acquire

- `path := <-wp.workers` (blocks until available)
- Returns absolute path to worktree directory

### Release

1. `git checkout <baseBranch>` (in worktree dir)
2. `git reset --hard <baseBranch>`
3. `git clean -fd`
4. `wp.workers <- path` (return to pool)

### Shutdown

1. For each worker path: `git worktree remove -f <path>`
2. `git worktree prune`
3. Remove `.expedition/worktrees/` directory

## CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--workers N` | 1 | Number of worktrees in pool |
| `--setup-cmd CMD` | (empty) | Command to run after worktree creation (e.g., `bun install`) |

### `--workers` behavior

| Value | Behavior |
|-------|----------|
| 0 | No worktree. Direct execution in Continent (full backward compat) |
| 1 | Single worktree. Sequential but isolated |
| 2+ | N worktrees pre-warmed. Sequential use today; parallel in future |

## Changes to Existing Code

### Modified

| File | Change |
|------|--------|
| `main.go` | Add `--workers`, `--setup-cmd` flags to Config |
| `paintress.go` | Create/shutdown pool in `Run()`. Acquire/Release around expedition + review |
| `expedition.go` | Add `WorkDir` field. Use `WorkDir` for `cmd.Dir` instead of `Continent` |

### Unchanged (uses Continent)

| File | Reason |
|------|--------|
| `lumina.go` | Journal scan reads `.expedition/journal/` at repo root |
| `flag.go` | Shared checkpoint at repo root |
| `devserver.go` | Single dev server, shared across all workers |
| `mission.go` | Written once at startup to repo root |

### Expedition Struct Extension

```go
type Expedition struct {
    // ...existing fields...
    Continent string   // repository root (journal, flag, etc.)
    WorkDir   string   // Claude Code execution directory (worktree or Continent)
}
```

`WorkDir` defaults to `Continent` when empty (backward compat for `--workers 0`).

## Filesystem Layout

```
<Continent>/
  .expedition/
    journal/          (shared - read by all workers)
    flag.md           (shared - checkpoint)
    lumina.md         (shared - learned patterns)
    mission.md        (shared - rules)
    worktrees/        (gitignored)
      worker-001/     (git worktree - isolated)
      worker-002/     (git worktree - isolated)
      worker-003/     (git worktree - isolated)
```

## Test Strategy

### testcontainers-go with alpine/git

All WorktreePool tests run inside Docker containers via testcontainers-go for complete environment isolation.

- **Image**: `alpine/git:latest` (~30MB, git only)
- **Pattern**: `containerGitExecutor` wraps `ctr.Exec()` to run git inside container
- **Lifecycle**: Container created per test (or per test suite), cleaned up automatically

### Test Cases

| Test | Validates |
|------|-----------|
| `TestInit_CreatesWorktrees` | `git worktree add` runs N times |
| `TestInit_PrunesStale` | Stale worktrees cleaned before creation |
| `TestInit_RunsSetupCmd` | Setup command executes in each worktree |
| `TestAcquire_ReturnsPath` | Channel returns valid path |
| `TestRelease_ResetsState` | Worktree is clean after release |
| `TestRelease_ReturnsToPool` | Released path is re-acquirable |
| `TestShutdown_RemovesAll` | All worktrees deleted, directory cleaned |

## Scope

### In Scope

- WorktreePool struct with Init/Acquire/Release/Shutdown
- GitExecutor interface (local + container implementations)
- CLI flags (--workers, --setup-cmd)
- Expedition.WorkDir field
- testcontainers-go test infrastructure

### Out of Scope

- goroutine parallel execution (future: Swarm Mode)
- sightjack integration (future: label-driven mission)
- Review loop worktree integration (next phase)
- Structured failure reporting (future: sightjack feedback)
