# Handover

**Last updated:** 2026-05-16 (asia/tokyo, Phase 1 complete)
**Updated by:** Claude Opus 4.7 session

## Current State

`feat/jun15-mcp-pivot` long-lived branch に **9 commit landed**
(origin push 済)。 refs/issues/0027 (jun15 MCP pivot v4) の Phase 1
MVP は **sub-A / sub-B / sub-C 全て完了**、 残作業は main merge 時の
PR 化と Phase 2 横展開のみ。

ADR 0017 (= docs/adr/0017-mcp-pivot.md) で architectural pin 確定:
human-initiated claude code session が LLM owner、 Go CLI は MCP
server (data/control plane) を提供する役割に縮約。

paintress repo の billing-boundary は完全 active:

- `.semgrep/jun15-no-headless-llm.yaml` 5 rule で headless LLM 経路を block、 transitional exclude なし (= production path に逃げ道なし)
- `internal/session/expedition.Run()` は `ErrMCPPivotDeprecated` を返す fail-fast stub、 LLM invocation block は完全除去
- MCP server (= `paintress mcp` subcommand) は 4 tool (= paintress.ping / next_issue / update_gradient / append_journal) を expose、 全 stub だが contract 固定済
- `/expedition-next` skill が plugins/paintress/skills/ 配下に存在、 claude code session から `--plugin-dir` + `--mcp-config` で driving 可能
- D-Mail 9-field envelope schema (= `internal/domain/dmail_envelope.go`) + synthetic fixture が固定、 sightjack 横展開時の cross-tool contract base

## In Progress

- branch: `feat/jun15-mcp-pivot` (push 済、 main merge 待ち)
- linked issue: `refs/HTMLification/docs/issues/0027-jun15-mcp-pivot.html`
- ADR: `docs/adr/0017-mcp-pivot.md` (= 本 commit で landed)
- Phase 1 MVP scope (= refs 0027 §8): **全 ✅ 完了**

### Phase 1 commit history (= 9 commits on feat/jun15-mcp-pivot)

| commit | type | scope |
|---|---|---|
| 5cfaef9 | chore(semgrep) | billing-boundary gate scaffold (5 rule + transitional exclude) |
| 9dcccd6 | feat(session) | MCP server stdio skeleton (paintress.ping) |
| b735e8c | feat(session) | real MCP tools stubs (next_issue / update_gradient / append_journal) |
| 15497e4 | feat(plugins) | /expedition-next skill definition |
| 3840915 | feat(domain) | D-Mail 9-field envelope schema + fixture + 5 contract test |
| 8249449 | refactor(session) | sub-A: Expedition.Run() deprecate stub (691 行削除 / 83 行追加) |
| 6817a39 | docs(handover) | sub-A progress 反映 |
| 958eae7 | refactor(session) | sub-B: semgrep transitional exclude 削除 + unused helper 削除 + 2 test 削除 |
| (本 commit) | docs(adr) | sub-C: ADR 0017 + handover finalize + 残 8 skipped test 完全削除 |

## Next Actions

1. **PR 化 + main merge** (= Phase 1 close、 immediate):
   - `feat/jun15-mcp-pivot` branch を GitHub PR として open
   - paintress CI (= `just lint-go` / `just semgrep` / `go test`) で green 確認
   - squash merge with ADR 0017 を merge commit に記録
2. **cost monitoring** (= 6/15 までの実測):
   - MCP tool invocation 数を OTel span に記録 (= 既存 platform.Tracer 経由)
   - `/expedition-next` slash command 実行回数を log 集計可能化
   - Anthropic dashboard で Agent SDK credit pool usage 0 を manual evidence として記録
3. **Phase 2** (= 6/15 以降継続):
   - sightjack に MCP pivot pattern を適用 (= paintress Phase 1 を copy)
   - amadeus / dominator も順次
   - phonewave は LLM 不使用なので変更なし (= D-Mail route daemon の role 維持)
   - cross-tool D-Mail contract を実 sightjack emit で end-to-end 検証

## Known Risks / Blockers

- `paintress run` 等の Go CLI 自動 expedition entry point は break、 operator は claude code session + `/expedition-next` skill 経由に移行必須 (= ADR 0017 §Consequences §Negative で明示)
- scheduler / CI で `paintress run` を wrap していた job は新運用に書き換え必要 (= human-in-the-loop step が入る)
- Phase 2 で sightjack を pivot する際、 paintress consume 側の MCP server に `paintress.consume_dmail` tool 追加が必要 (= 現状の stub では D-Mail consume path が未実装)
- Phase 1 で `BuildPrompt` / `loadInboxSection` / `loadContextSection` 等 helper を保留したが、 これらは MCP server 側で reuse する前提 (= 後続 commit で MCP tool 内 invocation 必要)

## Context the Next Actor Needs

- **canonical plan**: `refs/HTMLification/docs/issues/0027-jun15-mcp-pivot.html` (= v4、 codex 3 ラウンド review 反映済)
- **billing boundary 原則**: LLM 発火は常に human-initiated、 daemon は route まで、 consume 側は明示 slash command で trigger (= ADR 0017 §Decision 第 4 項)
- **semgrep gate**: `.semgrep/jun15-no-headless-llm.yaml` 5 rule、 production path に `permanent` nosemgrep 例外禁止 (= ADR 0017 §Decision 第 5 項)
- **D-Mail schema**: 9 field 固定、 file layout は `inbox/<message_id>.yaml + inbox/<message_id>.body.md` 2 file pair (= `internal/domain/dmail_envelope.go` + refs 0027 §8)
- **dotfiles plugins**: `~/dotfiles/plugins/paintress/` は試作品 sketch、 production target は paintress repo 内 `plugins/paintress/`

## Relevant Files and Commands

- `docs/adr/0017-mcp-pivot.md` - architectural pin (= 本 Phase 1 完了 marker)
- `.semgrep/jun15-no-headless-llm.yaml` - billing-boundary gate (5 rule)
- `internal/session/expedition.go` - Run() deprecate stub (= ErrMCPPivotDeprecated 返却)
- `internal/session/mcp_server.go` - JSON-RPC 2.0 stdio MCP server (4 tool advertised)
- `internal/cmd/mcp.go` - `paintress mcp` cobra subcommand
- `internal/domain/dmail_envelope.go` - 9-field cross-tool message schema
- `plugins/paintress/skills/expedition-next/SKILL.md` - human-driven entry point
- `tests/fixtures/dmail/dmail-2026-06-01T10-00-00Z-abc123.{yaml,body.md}` - contract test fixture
- `just lint-go` - golangci-lint v2 (= 0 issues 維持)
- `just semgrep` - semgrep gate (= 75 rules、 0 findings 維持)
- `go test -count=1 -timeout=120s ./internal/...` - Phase 1 test suite
- claude code 起動 (= ADR 0017 §Decision より):

  ```bash
  claude \
    --plugin-dir ./plugins/paintress \
    --mcp-config '{"paintress":{"command":"paintress","args":["mcp"]}}'
  ```
