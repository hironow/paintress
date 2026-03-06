# 0006. Bounded Concurrency with pond/v2

**Date:** 2026-02-25
**Status:** Accepted

## Context

`ScanJournalsForLumina` spawned one goroutine per journal file using
`sync.WaitGroup` + `sync.Mutex` for result collection. This created two
problems:

- **Unbounded goroutine count**: O(files) goroutines — with many journal files,
  this could exhaust memory or cause scheduler contention
- **No panic recovery**: A panic in any goroutine would crash the entire process
  without cleanup
- **Manual result collection**: `sync.Mutex` guarding `append()` into a shared
  slice is error-prone and obscures the data flow

The pattern was also inconsistent with `WorktreePool` (channel-based bounded
concurrency in `worktree.go`), creating two different concurrency models in the
same codebase.

## Decision

Replace unbounded goroutines with `pond/v2` `ResultPool[T]` bounded to
`runtime.GOMAXPROCS(0)` workers:

1. **Pool creation**: `pond.NewResultPool[journalData](runtime.GOMAXPROCS(0))`
   creates a worker pool sized to available CPU cores. Cleanup is deferred
   immediately: `defer pool.StopAndWait()`.

2. **Task submission**: `group.Submit(func() journalData { ... })` replaces
   `wg.Add(1)` + `go func()`. Each task returns a typed result directly.

3. **Result collection**: `group.Wait()` returns `([]journalData, error)`,
   eliminating the manual `sync.Mutex` + shared slice pattern. The error is
   non-nil only when a submitted task panics (defensive check for future
   changes).

4. **Dependency**: `github.com/alitto/pond/v2` added as a direct dependency
   (not indirect). pond was already a transitive dependency via
   `testcontainers-go`.

### Not applied

- **WorktreePool migration to pond**: `WorktreePool` uses a channel-based
  acquire/release pattern for reusable resources (git worktrees), which is
  fundamentally different from pond's fire-and-forget task model. The two
  patterns serve different purposes.

## Consequences

### Positive

- Goroutine count bounded: O(files) → O(cores)
- Built-in panic recovery prevents process crash from journal parsing errors
- Type-safe `group.Wait()` return eliminates manual mutex + slice pattern
- `defer pool.StopAndWait()` ensures cleanup on all code paths (including error)

### Negative

- New direct dependency (`pond/v2`) — though already transitively present
- Worker pool overhead for small file counts (negligible: pool creation is ~1μs)

### Neutral

- `GOMAXPROCS(0)` reads the current value without changing it (standard Go idiom)
- pond's `ResultPool` uses generics (`ResultPool[T]`), requiring Go 1.18+
  (paintress targets Go 1.26)
