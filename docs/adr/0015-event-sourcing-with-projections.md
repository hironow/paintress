# 0015. Event Sourcing with Projections and Replay

**Date:** 2026-03-05
**Status:** Accepted — supersedes 0014 (event sourcing without projections)

## Context

ADR 0014 adopted daily JSONL event sourcing for the expedition lifecycle but
explicitly noted "no projections or replay yet" as a neutral consequence.

Since then, both projection and replay capabilities have been implemented:

- `paintress rebuild` command replays all events from `.expedition/events/`
  to regenerate materialized projection state (`domain.EventApplier` +
  `session.ProjectionApplier`).
- Projection state tracks expedition counts (succeeded, failed, skipped)
  and gradient level.
- The `status` and `doctor` commands consume projection-derived data.

ADR 0014's characterization of the event store as "observational only" is
no longer accurate.

## Decision

Update the event sourcing architecture description to include projections:

- **Projection**: `domain.EventApplier` interface with `Rebuild([]Event) error`
- **Replay**: `paintress rebuild` command loads all events via `EventStore.LoadAll()`
  and applies them through the projector
- **Materialized state**: `session.ProjectionApplier` implements the interface,
  producing `ProjectionState` (TotalExpeditions, Succeeded, Failed, Skipped,
  GradientLevel)

The architecture is:

```
cmd → usecase → session → eventsource → domain
                  ↓
            ProjectionApplier (implements domain.EventApplier)
```

## Consequences

### Positive

- Documentation matches reality — rebuild, projections, and replay are implemented
- Clear separation: domain defines the EventApplier contract, session provides
  the concrete implementation
- Idempotent rebuild: can be run any time to regenerate consistent state

### Negative

- Projections are computed from all events on each rebuild (no incremental update)

### Neutral

- ADR 0014's other aspects (daily JSONL, fire-and-forget, 8 event types, pruning)
  remain unchanged
