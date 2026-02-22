# 0007. Per-Worker Flag Checkpoint with Reconciliation

**Date:** 2026-02-23
**Status:** Accepted

## Context

paintress supports concurrent expedition execution via `--workers N`, where
multiple goroutines run expeditions in parallel using git worktrees. Each
expedition writes a checkpoint (`flag.md`) recording the expedition number,
current issue, and status.

The initial implementation used a shared `Continent/.expedition/.run/flag.md`
protected by a mutex (`flagMu`). This caused cross-worker contamination: one
worker's `current_issue` would overwrite another's, and Claude Code's flag
watcher (`watchFlag`) would detect stale values from other workers (MY-362).

## Decision

Eliminate the shared checkpoint and mutex by giving each worker exclusive
ownership of its own `flag.md`:

1. **Workers=0** (single-threaded): `workDir` is empty, effective directory is
   Continent. Writes to `Continent/.expedition/.run/flag.md`.

2. **Workers>=1** (concurrent): `workDir` is the worktree path. Writes to
   `{worktree}/.expedition/.run/flag.md`. No two workers share a worktree.

3. **`reconcileFlags(continent, workers)`** (`paintress.go`): At startup, scans
   the Continent flag.md and (when workers > 0) all worktree flag.md files via
   `filepath.Glob`. Returns the one with `max(LastExpedition)` to determine the
   resume point. When workers == 0, worktree flags are skipped because
   `WorktreePool.Init` does not run, and leftover flags would be stale.

4. **Post-run consolidation**: After all workers complete (`g.Wait()`), the
   latest flag is written back to `Continent/.expedition/.run/flag.md` for
   human inspection.

5. **Stale clear** (`expedition.go`): Before starting the Claude process,
   `current_issue` from a previous interrupted expedition is cleared by
   re-writing flag.md without the `current_issue`/`current_title` fields.
   This operates on `workDir` (not Continent) to avoid clearing another
   worker's active state.

## Consequences

### Positive
- Mutex (`flagMu`) completely eliminated — no lock contention between workers
- Workers=0, Workers=1, and Workers>1 all use the same code path
- Each worker's flag watcher sees only its own state changes

### Negative
- Worktree cleanup failure could leave stale flag.md files (mitigated by
  `reconcileFlags`'s workers==0 guard that skips worktree scanning when
  `WorktreePool.Init` has not run)
- Post-run consolidation adds a small I/O operation at shutdown

### Neutral
- `flag.md` format is unchanged — only the write location changed
- `ReadFlag`/`WriteFlag`/`FlagPath` functions remain generic (take a base dir)
