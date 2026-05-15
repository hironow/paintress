# Handover

**Last updated:** 2026-05-15 (asia/tokyo, post sub-A)
**Updated by:** Claude Opus 4.7 session

## Current State

`feat/jun15-mcp-pivot` long-lived branch に **6 commit landed**
(origin に push 済)。 refs/issues/0027 (jun15 MCP pivot v4) の Phase 1
MVP は **sub-A 完了**、 残作業は sub-B (= semgrep transitional exclude
削除 + skipped test 完全削除) と sub-C (= ADR 0017 + handover
finalize)。

paintress repo の billing-boundary は機能している:

- `.semgrep/jun15-no-headless-llm.yaml` 5 rule で headless LLM 経路を block
- `internal/session/expedition.go` から `claude -p` invocation block (~335 行) を完全削除済
- production code 内で `Expedition.Run()` を呼ぶと `ErrMCPPivotDeprecated` が返却
- transitional exclude `internal/session/expedition.go` は yaml に残るが、 実コードからは LLM 呼び出しなし (= sub-B で exclude 自体を削除して機械的 enforce 化)
- MCP server (= `paintress mcp` subcommand) は 4 tool (paintress.ping / next_issue / update_gradient / append_journal) を expose、 全 stub だが contract 固定済

## In Progress

- branch: `feat/jun15-mcp-pivot`
- linked issue: `refs/HTMLification/docs/issues/0027-jun15-mcp-pivot.html`
- codex review: initial / v2 / v3 を経て v4 plan landed (refs main `5afdaf4`)
- Phase 1 MVP scope (= refs 0027 §8):
  - [x] feat/jun15-mcp-pivot branch 作成 + scaffold commit (5cfaef9)
  - [x] MCP server endpoint (= `internal/session/mcp_server.go`) skeleton + `paintress mcp` cobra subcommand (9dcccd6)
  - [x] paintress.next_issue / update_gradient / append_journal の MCP tool **interface fixed + stub** (b735e8c)
  - [x] `/expedition-next` slash command の skill definition (= `plugins/paintress/skills/expedition-next/SKILL.md`) (15497e4)
  - [x] synthetic D-Mail fixture contract test (= `internal/domain/dmail_envelope.go` 9-field schema + `tests/fixtures/dmail/*.yaml + .body.md` pair + 8 sub-test) (3840915)
  - [x] **sub-A**: `internal/session/expedition.go` の `claude -p` invocation を deprecate stub に置換 (= 691 行削除 / 83 行追加、 LLM invocation block ~335 行完全除去) (8249449)
  - [ ] **sub-B**: `.semgrep/jun15-no-headless-llm.yaml` の `internal/session/expedition.go` transitional exclude 削除 + `expedition_test.go` の 9 skipped test 完全削除 + nolint:unused 2 件削除 (= 完全 active な gate)
  - [ ] **sub-C**: `docs/adr/0017-mcp-pivot.md` 追加 (= architectural pin) + `docs/handover.md` finalize + plugins README finalize
  - [ ] OTel span で MCP tool invocation 数と slash command 実行回数を記録 (= refs 0027 §8 受入 b/c、 sub-C 以降で実装)

## Next Actions

1. **sub-B** (= Phase 1 完了 marker、 最 priority):
   - `.semgrep/jun15-no-headless-llm.yaml` から `internal/session/expedition.go` exclude 行を削除
   - `internal/session/expedition_test.go` の 9 skipped test (= TestLifecycle_Init/NoInit_Then_Expedition / TestExpedition_MidMatchedRouting_* 5 件 / TestExpedition_StaleFlagClearedOnWorkDir / TestExpedition_TwoWorkersConcurrent_NoContamination) を完全削除
   - `internal/session/expedition.go` の `//nolint:unused` 2 件 (= appendMidHighName / appendMidMatchedMail) と関連 struct field (midMatchedMu / midMatchedMails / midHighMu / midHighNames) を削除
   - `just semgrep` 0 finding 確認 (= transitional exclude なし state で gate 機能保証)
2. **sub-C** (= immutable pin):
   - `docs/adr/0017-mcp-pivot.md` 起票 (= 「LLM owner inversion / Go CLI を MCP server data plane に縮約」 という architectural decision を記録、 refs 0027 を ref として cite)
   - `docs/handover.md` を「Phase 1 全 commit landed、 main merge 待ち」 で finalize
   - `plugins/paintress/README.md` を「Phase 1 完了、 production target」 で finalize
3. **post-Phase 1**:
   - feat/jun15-mcp-pivot branch の PR 化 (= main merge)
   - cost monitoring 仕込み (= MCP tool invocation count を OTel span に追加、 slash command 実行回数を log 集計可能に)
   - Phase 2: sightjack / amadeus / dominator に MCP pivot pattern を順次横展開 (= refs 0027 §7 Phase 2 scope、 6/15 以降継続)

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
