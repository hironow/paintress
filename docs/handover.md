# Handover

**Last updated:** 2026-06-10 (JST)
**Updated by:** claude (AI draft from git history — review before trusting)

## Current State

Post-"jun15 MCP pivot": paintress is a Go CLI/MCP server acting as a pure data plane for the Expedition workflow (README). Extensive `docs/` exist (ADRs, dmail-protocol, conformance, policies, testing). Last meaningful commits (2026-06-10): "docs: add decision queue for human-review items (#252)" and "fix(sessions): keep mcp session wiring active" (8a064ab). Preceding work: MCP pivot doc/wording alignment (#243–#250), lint suppression cleanups (#237–#242), e2e migration to testcontainers-go (#235).

## In Progress

MCP pivot follow-ups (wording/doc alignment, #243–#250) appear recently completed; nothing else clearly in flight (git 履歴からは判別できず).

## Next Actions

1. requester による docs/intent.md ドラフトのレビューと確定
2. Work through `docs/decision-queue.md` human-review items (#252)

## Known Risks / Blockers

- The Agent SDK credit pool gates `claude --print` from 2026-06-15 (README) — this date is the stated motivation for the MCP pivot; verify nothing still depends on headless `claude --print` before then

## Context the Next Actor Needs

- The autonomous loop was retired; expedition runs now fire from a Claude Code session via the `/expedition-next` skill + paintress MCP tools (`plugins/paintress/skills/expedition-next/SKILL.md`)
- The MCP server runs over stdio: `paintress mcp` (embedded via `--mcp-config`)
- Tooling: Go + justfile + mise; CI: `ci.yaml`, `smoke-test.yaml`, `pr-title.yaml` (PR title check), `release.yaml`
- `docs/superpowers` is gitignored (#236)

## Relevant Files and Commands

- `README.md` — MCP pivot rationale and MCP tool list
- `docs/decision-queue.md` — pending human-review items
- `plugins/paintress/skills/expedition-next/SKILL.md` — workflow entry point
- `cmd/` / `internal/` — implementation
- `justfile` — build/lint/test recipes (`just help` for the list)
- `docs/conformance.md` / `docs/testing.md` — test expectations
