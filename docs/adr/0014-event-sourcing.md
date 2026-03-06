# 0014. Event Sourcing for Expedition Lifecycle

**Date:** 2026-02-27
**Status:** Superseded by [0015](0015-event-sourcing-with-projections.md)

## Context

ADR 0013 established a 2-layer architecture for paintress and explicitly
excluded event sourcing as YAGNI. However, the parent tap CLAUDE.md
mandates event sourcing across all four tools (phonewave, sightjack,
paintress, amadeus) as a core architecture principle.

sightjack uses single-file JSONL with sequence-based ordering. amadeus
uses daily JSONL rotation with timestamp-based queries. paintress is a
long-running daemon executing multiple expeditions over potentially many
days, making the amadeus pattern (daily rotation) the better fit.

## Decision

Adopt amadeus-style daily JSONL event sourcing:

- **EventStore** port interface in root package (`event.go`)
- **FileEventStore** adapter in `internal/eventsource/`
- **Daily files**: `continent/.expedition/events/YYYY-MM-DD.jsonl`
- **8 event types**: expedition.started, expedition.completed,
  dmail.staged, dmail.flushed, dmail.archived, gradient.changed,
  gommage.triggered, inbox.received
- **Fire-and-forget**: emit errors are logged, never propagated
- **Lifecycle**: FindExpiredEventFiles + PruneEventFiles for pruning

The architecture becomes 3-layer:

```
cmd → session → eventsource → root
```

Semgrep layer rules enforce the dependency direction.

## Consequences

### Positive

- Aligns paintress with parent tap architecture mandate
- Complete expedition lifecycle audit trail in immutable JSONL
- Daily rotation enables natural age-based pruning
- Fire-and-forget prevents event store from impacting daemon reliability

### Negative

- Adds `internal/eventsource/` layer (complexity increase)
- 3-layer diagram replaces simpler 2-layer model

### Neutral

- google/uuid dependency (already indirect) becomes direct
- Event store is observational only — no projections or replay yet
