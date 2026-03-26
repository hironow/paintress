# Gommage Recovery Strategy — Design Spec

## Problem

paintress Gommage fires when `consecutiveFailures >= 3`. The current behavior is uniform: always halt the expedition loop and escalate via D-Mail. This causes unnecessary downtime for transient failures (timeout, rate limit, parse error) that could self-heal with a retry after appropriate cooldown.

Additionally, when Gommage halts, the worktree is immediately destroyed — losing any partial progress (committed tests, partial implementation) that Claude made before the failure.

## Goals

1. Classify failure streaks by pattern (timeout, rate_limit, parse_error, blocker, systematic)
2. Retry the same issue in-place for transient classes, with appropriate cooldown
3. Preserve worktree during retry so partial progress survives
4. Resume from checkpoint after session restart (crash recovery)
5. Enforce a retry cap (2 attempts) to prevent infinite recovery loops
6. Enrich event data with class info for amadeus divergence scoring

## Non-Goals

- Changing the Gradient Gauge mechanics (separate topic: E12)
- Lumina pattern extraction improvements (separate topic: E11)
- Modifying the `maxConsecutiveFailures` threshold (stays at 3)

## Architecture: B+C Hybrid

Domain aggregate owns the classification and retry-count tracking (event-sourceable). Session layer owns the recovery side effects (cooldown, model switch, worktree lifecycle, Lumina injection).

```
+-------------------+     +-----------------------+
| ExpeditionAggregate|     | Paintress (session)   |
|                   |     |                       |
| DecideRecovery()  |---->| executeRecovery()     |
|   reasons []str   |     |   cooldown wait       |
|   recoveryAttempts|     |   model switch        |
|   -> Decision     |     |   lumina injection    |
|                   |     |                       |
| ResetRecovery()   |     | saveCheckpoint()      |
|   on success      |     | resumeIncomplete()    |
|                   |     | cleanOrphanWorktrees()|
+-------------------+     +-----------------------+

Legend:
- ExpeditionAggregate: domain layer, pure logic, event-sourced
- Paintress (session): I/O layer, side effects, worktree management
- DecideRecovery: classification + retry count + halt/retry verdict
- executeRecovery: class-specific cooldown + side effects
- saveCheckpoint: event recording of expedition progress
- resumeIncomplete: session startup resume from checkpoint
- cleanOrphanWorktrees: startup cleanup of leaked worktrees
```

## Failure Classes

| Class | Detection | Recovery | Cooldown (retry 1 / 2) |
|-------|-----------|----------|------------------------|
| `timeout` | `strings.Contains(reason, "timeout")` | Switch model + wait + retry same issue | 30s / 90s |
| `rate_limit` | `strings.Contains(reason, "rate_limit")` (marker injected by `handleExpeditionError`) | Wait + retry same issue | 60s / 180s |
| `parse_error` | `StatusParseError` in reason | Inject Lumina hint + retry same issue | 5s / 15s |
| `blocker` | `FailureType == "blocker"` in journal | Halt + escalate (current behavior) | N/A |
| `systematic` | 3+ identical reasons (default) | Halt + escalate (current behavior) | N/A |

Classification uses majority-vote: if 3+ of the last 5 reasons match a class keyword, that class wins. If no majority, `systematic` (safest: halt).

## Recovery Flow

```
Gommage fires (consecutiveFailures >= 3)
  |
  v
ClassifyGommage(reasons) -> class
  |
  v
DecideRecovery(reasons) -> RecoveryDecision
  |
  +-- Action: retry
  |     |
  |     v
  |   recoveryAttempts < maxRecoveryAttempts (2)?
  |     |
  |     +-- yes: executeRecovery()
  |     |         - class-specific side effect (model switch / lumina hint / noop)
  |     |         - save checkpoint (don't release worktree)
  |     |         - cooldown wait
  |     |         - reset consecutiveFailures
  |     |         - continue loop (retry same issue)
  |     |
  |     +-- no: fallback to halt
  |
  +-- Action: halt
        |
        v
      stageEscalation() (current behavior)
      releaseWorkDir()
      return errGommage
```

## Concurrency Safety

### Gommage Guard: CAS Instead of Load-then-Store

The existing TOCTOU race in the Gommage check (`Load` then `Store`) is fixed:

```go
// BEFORE (race-prone):
if p.consecutiveFailures.Load() >= threshold && !p.escalationFired.Load() {
    p.escalationFired.Store(true)

// AFTER (atomic):
if p.consecutiveFailures.Load() >= threshold && p.escalationFired.CompareAndSwap(false, true) {
```

Only one worker wins the CAS; others see `false` from `CompareAndSwap` and skip the block.

### Counter Reset Ownership

The authoritative `consecutiveFailures` counter is `p.consecutiveFailures` (session layer, `atomic.Int64`). The domain aggregate's `consecutiveFailures` is a shadow for event-sourced replay only. On recovery retry:

1. `p.consecutiveFailures.Store(0)` — session layer, gates re-trigger
2. Domain aggregate is NOT reset here (it tracks via events in `CompleteExpedition`)

This avoids split-brain: the session layer counter is the single source of truth for the Gommage guard.

### rate_limit Detection

`handleExpeditionError` injects a marker string into the journal reason when the reserve model is activated:

```go
if p.reserve.IsReserveActive() {
    reason = "rate_limit: " + reason
}
```

This makes `ClassifyGommage(reasons []string)` a true pure function — it only inspects reason strings, never session state.

## Worktree Lifecycle

### During Recovery Retry
- `releaseWorkDir()` is NOT called
- Worktree preserves committed and uncommitted changes
- Next retry starts Claude subprocess in the same worktree
- Claude receives `git log --oneline base..HEAD` + `git diff --stat` as context

### Cleanup Policy (3 layers)
1. **On retry success**: Normal expedition flow — PR created, worktree cleaned up after PR dispatch
2. **On retry cap reached (halt)**: `releaseWorkDir()` called as in current behavior
3. **On session startup**: `cleanOrphanWorktrees()` detects worktrees with stale checkpoints (>1h) and removes them

### Resume on Session Restart
1. `paintress run` checks event store for `expedition.checkpoint` events without a subsequent `expedition.completed`
2. Returns `[]IncompleteExpedition` — one per unfinished worker. Under `--workers=N`, up to N incomplete expeditions may exist
3. For each incomplete: if worktree still exists on disk and git state is valid, resume that expedition
4. Claude subprocess receives resume context: `git log --oneline` + `git diff --stat`
5. Claude reads files as needed via its `Read` tool (lightweight context injection)
6. Expeditions that cannot be resumed (missing worktree, corrupted git) are logged and their worktrees cleaned up

## Domain Types

### `GommageClass` (new type in `domain/gommage_classifier.go`)
```go
type GommageClass string
// Constants: GommageClassTimeout, GommageClassRateLimit,
// GommageClassParseError, GommageClassBlocker, GommageClassSystematic
```

### `RecoveryDecision` (new type in `domain/gommage_recovery.go`)
```go
type RecoveryDecision struct {
    Action      RecoveryAction  // "retry" or "halt"
    Class       GommageClass
    Cooldown    time.Duration
    RetryNum    int
    MaxRetry    int
    KeepWorkDir bool
}
```

### `ExpeditionAggregate` changes
- New field: `recoveryAttempts int` — scoped to current failure streak, resets when `consecutiveFailures` resets (on any success). In-memory only, not event-sourced (derived from streak state).
- New method: `DecideRecovery(reasons []string) RecoveryDecision`
- New method: `ResetRecovery()` — called when `consecutiveFailures` resets (success or manual reset). Clears `recoveryAttempts` to 0.

### Event data extensions
- `GommageTriggeredData`: adds `Class`, `RecoveryAction`, `RetryNum` (all `omitempty`)
- New: `GommageRecoveryData` for `gommage.recovery` event
- New: `ExpeditionCheckpointData` for `expedition.checkpoint` event

## Session Types

### `gommage_recovery.go`
- `executeRecovery(ctx, decision, exp, expedition) bool` — returns true to retry
- `injectParseErrorLumina(continent, logger)` — writes corrective Lumina hint

### `gommage_checkpoint.go`
- `saveCheckpoint(exp, phase, workDir)` — records progress event
- `resumeIncompleteExpeditions() []IncompleteExpedition` — startup resume (returns all incomplete, one per worker)
- `buildResumeContext(workDir string) string` — `git log --oneline` + `git diff --stat`
- `cleanOrphanWorktrees()` — startup cleanup

## Emitter Interface Extensions

```go
EmitGommageRecovery(expedition int, decision RecoveryDecision, now time.Time) error
EmitCheckpoint(expedition int, phase, workDir string, commitCount int, now time.Time) error
```

## OTel Observability

Gommage span event gains: `gommage.class`, `gommage.action`, `gommage.retry_num`.
Recovery span event: `gommage.recovery` with class and cooldown attributes.

## Negative Feedback Properties

The design maintains a negative feedback regime:
- Transient failures self-heal without human escalation
- Recovery cap (2 retries) prevents runaway retry loops
- Each successful recovery reduces future Gommage probability (counter reset)
- Persistent failures still halt and escalate
- amadeus can audit recovery patterns via enriched event data

## File Changes

| Action | File | Description |
|--------|------|-------------|
| NEW | `domain/gommage_classifier.go` | ClassifyGommage pure function |
| NEW | `domain/gommage_recovery.go` | RecoveryAction, RecoveryDecision types, cooldownForClass |
| NEW | `session/gommage_recovery.go` | executeRecovery, injectParseErrorLumina |
| NEW | `session/gommage_checkpoint.go` | checkpoint, resume, orphan cleanup, buildResumeContext |
| MOD | `domain/expedition_aggregate.go` | recoveryAttempts, DecideRecovery, ResetRecovery |
| MOD | `domain/event.go` | GommageTriggeredData extension, 2 new event types + data |
| MOD | `session/paintress_expedition.go` | L215-236 class-aware dispatch |
| MOD | `session/paintress.go` | startup orphan cleanup + resume check |
| MOD | `session/gommage_insight.go` | Class parameter |
| MOD | emitter port interface | 2 new methods |

## Test Plan

| Layer | File | Cases | Count |
|-------|------|-------|------:|
| domain | `gommage_classifier_test.go` | Timeout, RateLimit, ParseError, Blocker, Systematic, Mixed, Empty | 7 |
| domain | `expedition_aggregate_test.go` | RetryOnTimeout, RetryOnRateLimit, HaltOnBlocker, HaltAfterMaxRetries, ResetOnSuccess | 5 |
| domain | `gommage_recovery_test.go` | CooldownForClass x3, Serialization | 4 |
| session | `gommage_recovery_test.go` | RetryTimeout, RetryRateLimit, HaltSystematic, ContextCancelled | 4 |
| session | `gommage_checkpoint_test.go` | SaveCheckpoint, ResumeExists, ResumeMissing, ResumeCorrupted, CleanOrphan, BuildResumeContext | 5 |
| integration | `paintress_test.go` | TimeoutThenSuccess, MaxRetriesThenHalt | 2 |
| **Total** | | | **27** |

All tests follow TDD: RED (failing test) -> GREEN (minimal impl) -> REFACTOR.
