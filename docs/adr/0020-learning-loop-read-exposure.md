# 0020. Learning-loop read exposure (get_insights)

**Date:** 2026-06-10
**Status:** Accepted

## Context

The jun15 MCP pivot retired the headless expedition loop, and with it
every caller of the Lumina learning surface: `ScanJournalsForLumina`
and `WriteLuminaInsights` had zero production callers (verified
2026-06-10, refs issue 0034). The learn stage of the self-improvement
loop was invisible to the Claude Code session.

## Decision

1. **`get_insights` read-only MCP tool**: returns the persisted
   insight-ledger files (`.expedition/insights/*.md`, parsed by the
   existing `InsightWriter.Read`) plus a **live Lumina scan** —
   `ScanJournalsForLumina` recomputed from the journals on every call.
2. **No write path is introduced**: live recomputation makes the
   dormant `WriteLuminaInsights` unnecessary for freshness; the journal
   files written by `append_journal` are the durable learning input.
   Read-only, always fresh, idempotent.
3. **Empty state is not an error**: missing files / no journals return
   empty arrays with an instruction string.
4. **Skill v0.3.1**: `/expedition-next` consults `get_insights` before
   implementing (defensive patterns = do not repeat; success patterns =
   proven approaches).

## Consequences

### Positive

- The learning loop closes for the implementer: journals written by
  past expeditions shape the next one, through an audited read surface.
- No new persistence, no idempotency hazards, no port wiring.

### Negative

- Live scan cost grows with journal count (bounded: parallel scan,
  10-lumina cap; acceptable at current scale).

### Neutral

- `WriteLuminaInsights` / gommage insight writers remain dormant; a
  future wave may retire or revive them once real usage data exists.
