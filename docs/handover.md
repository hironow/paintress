# Handover

**Last updated:** 2026-05-15 (asia/tokyo)
**Updated by:** Claude Opus 4.7 session

## Current State

`feat/jun15-mcp-pivot` long-lived branch を切り、 refs/issues/0027
(jun15 MCP pivot v4) の Phase 1 MVP scaffold を入れた。 paintress repo
の semgrep gate (`.semgrep/jun15-no-headless-llm.yaml`) が 0 finding で
動作する。 既存 `internal/session/expedition.go` の `claude -p`
invocation は transitional exclude path で一時許容、 後続 commit で
MCP server pivot 完了時に削除する。

## In Progress

- branch: `feat/jun15-mcp-pivot`
- linked issue: `refs/HTMLification/docs/issues/0027-jun15-mcp-pivot.html`
- codex review: initial / v2 / v3 を経て v4 plan landed (refs main `5afdaf4`)
- Phase 1 MVP scope (= refs 0027 §8):
  - [x] feat/jun15-mcp-pivot branch 作成 + scaffold commit
  - [ ] MCP server endpoint (= `internal/adapter/input/mcp/server.go`) skeleton
  - [ ] paintress.next_issue / update_gradient / append_journal の MCP tool 定義
  - [ ] `/expedition next` slash command (= `plugins/paintress/skills/` 経由)
  - [ ] synthetic D-Mail fixture contract test (= inbox に YAML fixture を置いて consume)
  - [ ] `internal/session/expedition.go` の `claude -p` invocation を MCP に置換、 semgrep exclude path を削除
  - [ ] OTel span で MCP tool invocation 数と slash command 実行回数を記録

## Next Actions

1. MCP server endpoint skeleton を `internal/adapter/input/mcp/server.go` に
   配置 (= 雛形、 tool 定義は別 commit)
2. paintress domain の event sourcing / gradient gauge / lumina 状態を
   MCP resource として expose する design 検討
3. `/expedition next` slash command の skill definition 雛形を
   `plugins/paintress/skills/expedition/SKILL.md` に書く
4. synthetic D-Mail fixture を `tests/fixtures/dmail/` に配置 (= refs 0027
   §8 の 9-field YAML schema を follow)
5. `expedition.go` の MCP 移行と semgrep exclude 削除を 1 PR で実施

## Known Risks / Blockers

- semgrep test framework の rule id mismatch (= `--test` mode で fixture が
  scan される前に exclude される問題) は解決済、 fixture file を
  Phase 1 scaffold では同梱せず production scan の 0 finding で代用。
  後続 commit で fixture refine が必要。
- Anthropic Agent SDK credit pool usage 0 の machine-readable 確認は
  公式 endpoint が未証明、 §8 で任意 manual evidence に格下げ済。
- 6/15 までに paintress MVP のみ完成が現実的、 残 4 tools (sj/am/dom/pw)
  は credit pool で当面回す + cost monitoring で持続性実測。

## Context the Next Actor Needs

- canonical plan は `refs/HTMLification/docs/issues/0027-jun15-mcp-pivot.html`
  のみ、 `/tmp/jun15-pivot-plan-*.md` は throwaway。
- LLM 発火は **必ず human-initiated** (= 明示 slash command)、 SessionStart
  hook 等の自動起動 + hook stdout への payload 出力は禁止 (= refs 0027 §5
  原則 1 + 原則 2)。
- semgrep rule の `[permanent]` 例外は本 pivot rule に対しては禁止、
  `[expires: YYYY-MM-DD]` 期限付きのみ許可。
- D-Mail 最小 schema (9 field) は refs 0027 §8 で固定済、 sightjack
  横展開時に paintress consume 契約を壊さないための base。

## Relevant Files and Commands

- `.semgrep/jun15-no-headless-llm.yaml` — billing boundary gate (= Phase 1
  acceptance criteria の中核)
- `internal/session/expedition.go` — transitional exclude 対象、 MCP
  pivot で `-p prompt` arg pair を削除予定
- `refs/HTMLification/docs/issues/0027-jun15-mcp-pivot.html` — canonical plan
- `just semgrep` — 0 finding 確認 (= production scan で billing boundary
  enforce)
- `just lint` — ruff + mypy 等 (= 既存 quality gate)
- `just test` — go test (= 既存 quality gate)
