# 0019. MCP dmail emission and project wiring

**Date:** 2026-06-10
**Status:** Accepted

## Context

The jun15 MCP pivot (ADR 0017/0018) left paintress with working write
tools (update_gradient / append_journal) but no sanctioned D-Mail
emission path (`SendDMail` survived orphaned) and no distribution
mechanism for the entry skill (refs issue 0032; zero invocations to
date). Claude Code conformance constraints C1-C6 (refs issue 0032 §5)
bound the design.

## Decision

1. **Dot-free tool names** (C1): `paintress.ping` → `ping` etc.
2. **`dmail` emission tool** (refs issue 0031): typed D-Mail v1 subset
   built by `domain.NewProducedDMail` (producer kind: report), sent
   through the existing `SendDMail` path — SQLite stage → atomic flush
   plus dmail.staged/flushed events via the wired expedition emitter.
3. **Project wiring** (C4/C5, decision D5(a)): `init` materializes the
   entry skill into the project's `.claude/skills/`; `mcp-config
   generate` upserts the project-root `.mcp.json` merge-aware. The
   canonical-locked `mcp_config.go` stays byte-identical (upsert in
   `claude_wiring.go`, wired from cmd).
4. **`instructions` in the initialize handshake** (C6).
5. **Skill v0.3.0**: wave-mode issue source (spec D-Mail inbox), report
   emission via `dmail`, `disable-model-invocation: true`.

## Consequences

### Positive

- The implementer loop closes durably: spec in → implement → journal →
  report out, all through audited paths.
- Zero-flag session startup in initialized projects.

### Negative

- Renaming tools is a wire-contract break (accepted while invocations
  are zero).

### Neutral

- `plugins/` becomes a pointer README; the skill's canonical source
  lives under `internal/platform/templates/claude-skills/`.
