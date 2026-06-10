# Intent

**Last updated:** 2026-06-10
**Requester:** hironow
**Status:** DRAFT — AI が README / git 履歴から起草。requester 未確認
**Work unit:** paintress — Go MCP server + data plane for the Expedition workflow

## Goal

Provide an MCP server and pure data plane for the Expedition workflow: serve the expedition journal/gradient read models from the session's continent dir and persist expedition-completed / gradient events (README). Following the "jun15 MCP pivot", LLM ownership moved to a human-initiated Claude Code session; the headless autonomous loop (`claude --print` subprocess, swarm worktree pool, review gate, gommage recovery, D-Mail composition) has been retired.

## Success Criteria

- CI workflows green (`ci.yaml`, `smoke-test.yaml`, `pr-title.yaml`, `release.yaml`)
- Tests pass via the justfile recipes (`tests/` exists; e2e migrated to testcontainers-go in #235)
- 上記以外の成果基準は未定義 — Open Questions 参照

## Scope

### In scope

- `paintress mcp` stdio server exposing `paintress.ping` / `paintress.next_issue` / `paintress.update_gradient` / `paintress.append_journal` (README)
- Data-plane commands: init, doctor, status, sessions, archive-prune, ... (README)
- Event store persistence for gradient + expedition-completed events
- Claude Code plugin/skills under `plugins/paintress/` (e.g. the `expedition-next` skill)

### Out of scope (Non-goals)

- Driving the LLM loop itself or composing D-Mails — explicitly retired in the MCP pivot (README)

## Constraints

- Inference must stay on the Claude Code session's subscription quota, not the Agent SDK credit pool that gates `claude --print` from 2026-06-15 (README)
- Go toolchain; tasks via justfile; tool versions via mise

## Open Questions

- [ ] requester による本ドラフトのレビュー
- [ ] Whether any pre-pivot code paths remain to be removed after the MCP pivot wording/doc alignment commits (#243–#250)
- [ ] Handling of `docs/decision-queue.md` human-review items (#252)
