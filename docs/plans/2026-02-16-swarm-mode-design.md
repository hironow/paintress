# Swarm Mode Design

**Goal:** Parallelize expedition execution using goroutine workers, each running in an isolated worktree.

**Architecture:** Fan-out N worker goroutines from `Paintress.Run()`, managed by `errgroup`. Each worker independently loops: claim expedition number → acquire worktree → run expedition → process result → release worktree. Shared state synchronized via `sync/atomic` and `sync.Mutex`.

**Key constraint:** Expedition is unique and self-contained. The only change is the execution environment (worktree). Expedition logic itself is untouched.

---

## Design Decisions

### Parallelism Model
- `--workers N` = goroutine count = worktree count (1:1 mapping)
- `--workers 0` = direct execution (1 goroutine, no worktree pool)
- `--workers 1` = 1 worktree, 1 goroutine (same code path as N > 1)

### Shared State Synchronization

| State | Current | Swarm Mode | Mechanism |
|-------|---------|------------|-----------|
| totalSuccess/Failed/Skipped/Bugs | `int` | `atomic.Int64` | Lock-free counter |
| consecutiveFailures | `int` | `atomic.Int64` | `Add(1)` on fail, `Store(0)` on success |
| Expedition numbering | computed from counters | `atomic.Int64` | `Add(1)-1` for fetch-increment |
| Flag checkpoint | `WriteFlag()` | `sync.Mutex` | Caller-side lock |
| WriteLumina | `WriteLumina()` | `sync.Mutex` | Caller-side lock |
| GradientGauge | `sync.RWMutex` | no change | Already thread-safe |
| ReserveParty | `sync.RWMutex` | no change | Already thread-safe |
| WorktreePool | `chan string` | no change | Already thread-safe |

### Gommage (Consecutive Failure Detection)
- Global count across all workers (expedition-level, not worker-level)
- Any success from any worker resets counter to 0
- Race between `Add(1)` and `Store(0)` is acceptable: worst case is false-positive gommage (safe side)
- Gommage triggers `errGommage` sentinel error → errgroup cancels all workers

### Review Gate
- Each worker independently runs review loop after successful expedition
- Multiple PRs can be reviewed concurrently
- No serialization needed (review operates on isolated worktree)

### Lumina Scan
- Runs once before worker launch (not per-expedition in parallel mode)
- Lumina slice passed to workers as read-only data
- WriteLumina protected by mutex (called from orchestrator only at startup)

---

## Structural Changes

### Paintress struct additions
```go
type Paintress struct {
    // ... existing fields unchanged ...

    // Swarm Mode: atomic counters
    expCounter          atomic.Int64
    totalSuccess        atomic.Int64
    totalSkipped        atomic.Int64
    totalFailed         atomic.Int64
    totalBugs           atomic.Int64
    consecutiveFailures atomic.Int64

    // Swarm Mode: mutex-protected resources
    flagMu   sync.Mutex
    luminaMu sync.Mutex
}
```

### Run() orchestration
```
Run():
  1. Init (log, mission, banner, devserver, pool)
  2. Lumina scan (once)
  3. Read Flag, init expCounter
  4. Launch N workers via errgroup
  5. Wait for all workers
  6. Print summary, return exit code
```

### runWorker() loop
```
runWorker(ctx, workerID, luminas):
  loop:
    1. Check ctx.Done()
    2. Claim expedition number (atomic)
    3. Check maxExpeditions budget
    4. Acquire worktree
    5. Build & run expedition
    6. Process result (atomic counters, mutex flag)
    7. Review loop (if success + PR)
    8. Check gommage → return errGommage
    9. Check StatusComplete → return errComplete
   10. Release worktree
   11. Cooldown (10s)
```

### Sentinel errors
```go
var (
    errGommage  = errors.New("gommage: consecutive failures exceeded")
    errComplete = errors.New("all issues complete")
)
```

---

## Files Changed

| File | Change |
|------|--------|
| `paintress.go` | struct (atomic/mutex), `Run()` → errgroup, new `runWorker()` |
| `go.mod` | add `golang.org/x/sync` (errgroup) |
| `paintress_test.go` (new) | Swarm Mode integration tests |

**All other files unchanged.** Existing thread-safe designs (GradientGauge, ReserveParty, WorktreePool) require no modification.
