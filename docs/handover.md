# Handover

**Last updated:** 2026-05-22 (asia/tokyo, 0028 residue cleanup landed)
**Updated by:** Claude Opus 4.7 session

## Current State

jun15 MCP pivot (refs/issues/0027) **全 phase 完了 + archive 入り**、
**かつ 0028 helper-level residue cleanup も完了**。
`feat/jun15-mcp-pivot` long-lived branch から始まり、 Phase 2-4 の
follow-up + 0028 helper-level stub 含めて **5 ツール共通の MCP
server-first architecture が main merged** で確立。

paintress 固有の jun15 landmark:

- ADR 0017 (= `docs/adr/0017-mcp-pivot.md`) で architectural pin 固定
- ADR 0018 (= `docs/adr/0018-mcp-pivot-helper-level-stub.md`) で
  helper-level enforcement extend (= ClaudeAdapter / doctor probe /
  issues cobra の stub 完了、 0017 の §Enforcement inventory gap 補強)
- 4 MCP tool 全 real impl + emitter wiring 済 (= ping / next_issue /
  update_gradient / append_journal)
- Phase 4 #4 (PR #218 `03afd1e`): `port.ExpeditionEventEmitter` を cmd
  composition root で構築し session に注入、 update_gradient →
  EventGradientChanged / append_journal → EventExpeditionCompleted を
  event store に自動 emit
- 0028 residue cleanup: `ClaudeAdapter.RunDetailed` 280 行を
  ErrMCPPivotDeprecated 返却 stub に圧縮、 `doctor.go` の
  claude-inference probe を unconditional CheckSkip、
  `paintress issues` cobra を fast-fail (= 副作用ゼロで deprecation
  message)
- `.semgrep/jun15-no-headless-llm.yaml` **6 rule** (= 0017 base 5 +
  0018 追加 `jun15-no-print-flag-literal-go`) で headless LLM 経路 +
  dynamic args spread を permanent block (= 5 ツール symmetric)
- `internal/session/expedition.Run()` は `ErrMCPPivotDeprecated` を返す
  fail-fast stub のまま、 LLM invocation block は完全除去
- `/expedition-next` skill (`plugins/paintress/skills/`) が claude code
  session 経由の唯一の expedition driver
- D-Mail 9-field envelope schema (`internal/domain/dmail_envelope.go`)
  + cross-tool fixture が cross-repo contract base として固定

## In Progress

なし。 jun15 MCP pivot + 0028 residue cleanup ともに完了。 refs 0027
は archive、 0028 は archive 待ち (= 本 PR merge 後)。

## Next Actions

なし (= Phase 4 #1-#4 全完了)。 後続作業候補は別 issue で fork:

1. **dominator EventJudgmentRecorded 拡張**: paintress Phase 4 #4 と同
   pattern で dominator.record_result を preview-only → event store
   emit へ昇格 (= scope やや中規模)
2. **Phase 3 cost (c) Anthropic dashboard credit 0 verify**: jun15
   launch (2026-06-15) 以降の operational evidence 収集 (= manual)

## Known Risks / Blockers

- `paintress run` 等の Go CLI 自動 expedition entry point は依然 break、
  operator は claude code session + `/expedition-next` skill 経由必須
  (= ADR 0017 §Consequences §Negative で明示)
- scheduler / CI で `paintress run` を wrap していた job は新運用書き換え必要
- 旧 worktree-based e2e test (= `internal/session` 全体 timeout する
  test) は `-short` flag で skip 推奨 (= 既知の Docker 依存)

## Context the Next Actor Needs

- **canonical plan archive**: `tap/refs/HTMLification/docs/archive/0027-jun15-mcp-pivot.html`
  (= immutable、 status pill `✅ ARCHIVED`)
- **post-mortem**: `tap/refs/HTMLification/lessons/0027-jun15-mcp-pivot-post-mortem.html`
  (= 7 reusable patterns catalog)
- **billing boundary 原則**: LLM 発火は常に human-initiated、 daemon は route まで、
  consume 側は明示 slash command で trigger (= ADR 0017 §Decision 第 4 項)
- **semgrep gate**: `.semgrep/jun15-no-headless-llm.yaml` 5 rule、 production
  path に `permanent` nosemgrep 例外禁止 (= ADR 0017 §Decision 第 5 項)
- **D-Mail schema**: 9 field 固定、 file layout は `inbox/<message_id>.yaml +
  inbox/<message_id>.body.md` 2 file pair
- **port-adapter 境界**: aggregate / emitter 構築は cmd composition root、
  session 直接 new は `session-no-direct-new-aggregate` semgrep rule で禁止

## Relevant Files and Commands

- `docs/adr/0017-mcp-pivot.md` - architectural pin (Phase 1 完了 marker)
- `docs/adr/0018-mcp-pivot-helper-level-stub.md` - 0017 §Enforcement
  inventory の helper-level gap 補強 ADR (= 0028 residue cleanup の
  rationale + acceptance criteria を記録)
- `.semgrep/jun15-no-headless-llm.yaml` - billing-boundary gate
  (6 rule、 0018 で `jun15-no-print-flag-literal-go` 追加)
- `internal/session/claude_adapter.go` - ErrMCPPivotDeprecated 即返却
  stub (= 0018 で 280 行 → 60 行に圧縮)
- `internal/session/mcp_server.go` - JSON-RPC 2.0 stdio MCP server (4 tool real impl + emitter wiring)
- `internal/cmd/mcp.go` - `paintress mcp` cobra subcommand (= EventStore + ExpeditionAggregate + ExpeditionEventEmitter を構築して session に注入)
- `internal/domain/dmail_envelope.go` - 9-field cross-tool message schema
- `plugins/paintress/skills/expedition-next/SKILL.md` - human-driven entry point
- `just lint-go` - golangci-lint v2 (0 issues 維持)
- `just semgrep` - semgrep gate (75 rules、 0 findings 維持)
- `go test -short -count=1 -timeout=120s ./internal/...` - Phase 1-4 test suite
- claude code 起動 (= ADR 0017 §Decision より):

  ```bash
  claude \
    --plugin-dir ./plugins/paintress \
    --mcp-config '{"paintress":{"command":"paintress","args":["mcp"]}}'
  ```
