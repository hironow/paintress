# 0012. Defer-Unlock Enforcement via Semgrep

**Date:** 2026-02-25
**Status:** Accepted

## Context

The codebase had 7 instances of `mu.Lock()` without `defer mu.Unlock()`,
spread across `devserver.go` (4 sites) and `expedition.go` (3 sites). These
used the Lock/set/Unlock pattern:

```go
ds.mu.Lock()
ds.running = true
ds.mu.Unlock()
```

This pattern is error-prone: adding a `return` or `panic` between Lock and
Unlock causes a deadlock. Code reviewers must manually verify every Lock/Unlock
pair, which does not scale.

ADR 0005 (fsnotify daemon) established `defer watcher.Close()` as a convention,
but the defer-unlock principle was not codified as a codebase-wide rule.

## Decision

Mandate `defer mu.Unlock()` immediately after every `mu.Lock()` call and
enforce via semgrep static analysis:

1. **Method extraction pattern**: When Lock/set/Unlock guards a single field
   access, extract a dedicated method that encapsulates the critical section:

   - `DevServer`: `isRunning() bool` and `setRunning(v bool)` replace 4 inline
     Lock/set/Unlock sites
   - `Expedition`: `appendMidHighName(name string)` and
     `appendMidMatchedMail(dm DMail)` replace 2 inline sites in callbacks
   - `Expedition`: `setCurrentIssue(issue string)` changed from Lock/set/Unlock
     to Lock/defer-Unlock

2. **Semgrep rule** (`adr0005-mutex-lock-without-defer-unlock` in
   `.semgrep/adr.yaml`): Detects `$M.Lock()` not followed by
   `defer $M.Unlock()` in the same scope. Severity: ERROR.

3. **Test file exclusion**: `*_test.go` is excluded from the rule because test
   files legitimately use manual Lock/Unlock for assertions (e.g.,
   `inbox_watcher_test.go` with 19 Lock sites, `race_test.go`).

4. **Escape hatch**: When conditional Unlock is truly required (e.g., a
   Lock-check-Unlock-else-continue pattern), annotate with `// nosemgrep` and
   a justification comment. No such cases currently exist in the codebase.

### Not applied

- **RWMutex splitting**: `DevServer.mu` protects both reads and writes to
  `running`, but the lock is held for < 1μs. RWMutex overhead exceeds benefit.
  `ReserveParty` already uses `sync.RWMutex` where read-heavy access justifies
  it (ADR 0009).

## Consequences

### Positive

- Deadlock prevention — every Lock is guaranteed to Unlock via defer, even on
  panic or early return
- Static enforcement — semgrep catches violations at `just semgrep` time, before
  code review
- Method extraction improves readability — callers no longer touch mutexes
  directly (e.g., `e.appendMidHighName(dm.Name)` vs inline Lock/append/Unlock)
- 7 findings → 0 findings after migration

### Negative

- Method extraction adds indirection for trivial operations (mitigated by Go's
  inlining — these methods are small enough to be inlined by the compiler)
- Semgrep's pattern matching cannot distinguish truly safe manual Unlock from
  bugs, requiring `nosemgrep` for legitimate exceptions

### Neutral

- `DevServer.Stop()` already used `defer ds.mu.Unlock()` before this decision
  — the rule codifies the existing best practice
- The semgrep rule ID references ADR 0005 (`adr0005-*`) for historical
  continuity, but this ADR (0012) is the canonical source for the defer-unlock
  mandate
