# 0006. Game-Inspired Expedition System

**Date:** 2026-02-23
**Status:** Accepted

## Context

paintress is an autonomous code implementation tool that orchestrates Claude Code
to pick Linear issues, implement solutions, and create pull requests. This
requires managing state transitions (issue selection, implementation, review),
failure recovery, model switching on rate limits, and adaptive difficulty scaling.

Using generic names (e.g., "run", "cycle", "config") for these interconnected
concepts would scatter the metaphor and make onboarding new developers harder.
A cohesive domain vocabulary naturally defines module boundaries and makes the
system's behavior intuitive from the naming alone.

## Decision

Adopt a game-inspired (MMO/RPG) metaphor across the entire codebase:

1. **Continent** (`Config.Continent`): The target repository's working directory.
   Like a game world map, it defines the scope of exploration.

2. **Expedition** (`expedition.go`): A single Claude Code execution cycle
   comprising issue selection, implementation, and PR creation. Each expedition
   is numbered sequentially and produces a journal entry.

3. **Gradient Gauge** (`gradient.go`): A momentum/combo meter tracking
   consecutive successes on a 0-5 scale. Success charges (+1), failure
   discharges to 0, skip decays (-1). At max level, a "Gradient Attack"
   unlocks the highest priority issues. The gauge is injected into prompts
   to guide Claude Code's issue selection.

4. **Reserve Party** (`reserve.go`): Model fallback on rate limits. When the
   primary model (e.g., Opus) hits rate limits, the reserve model (e.g.,
   Sonnet) steps in automatically with a 30-minute cooldown before primary
   recovery. Named after the RPG pattern where reserve party members replace
   fallen active members.

5. **Lumina** (`lumina.go`): Aggregated lessons extracted from expedition
   journals. Past expeditions produce knowledge that illuminates future ones,
   injected into prompts as guidance patterns.

6. **D-Mail** (`dmail.go`): Asynchronous inter-tool messages exchanged through
   directory-based mailboxes (inbox/outbox/archive). The metaphor comes from
   "Divergence Mail" — messages that cross tool boundaries.

7. **Journal** (`journal.go`): Post-expedition reports stored in
   `.expedition/journal/NNN.md`. Each expedition records its outcome, forming
   the raw material for Lumina extraction.

## Consequences

### Positive
- Metaphor consistency makes conceptual mapping intuitive — "Reserve Party
  activated" immediately conveys "fallback model engaged due to rate limit"
- Domain vocabulary naturally defines module boundaries (one file per concept)
- Game terminology creates memorable, distinct names that avoid collision with
  Go standard library or common programming terms

### Negative
- New developers must learn the metaphor vocabulary before understanding the
  codebase (mitigated by `docs/` documentation and inline comments)
- Non-gamers may find the naming unfamiliar (mitigated by the comment on each
  type explaining the real-world mapping)
