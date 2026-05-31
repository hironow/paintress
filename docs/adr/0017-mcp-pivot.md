# 0017. MCP pivot: claude code session owns LLM, paintress Go CLI is MCP server data plane

**Date:** 2026-05-15
**Status:** Accepted

## Context

Starting 2026-06-15, Anthropic Claude Code subscription plans (Pro,
Max 5x, Max 20x) bill `claude -p` and Agent SDK usage against a
separate monthly Agent SDK credit pool ($20 / $100 / $200) that is
disjoint from the interactive usage quota. The previous paintress
Go CLI architecture invoked `claude -p` as an exec.Command subprocess
on every expedition, which means every production run after that date
would draw on credit pool capacity that is not sized for autonomous
TDD loops.

We surveyed every technically plausible way to keep the existing
control flow running off the interactive quota:

- PTY automation (`creack/pty`, `expect`), tmux `send-keys` +
  `capture-pane`, Remote Control protocol use, and TTY spoofing via
  `script(1)` all violate the Anthropic Acceptable Use Policy clause
  on bypassing product-imposed restrictions, regardless of intent.
- `--output-format`, `--input-format`, `--fallback-model`,
  `--max-budget-usd`, and the rest of the structured-output flag set
  are documented as `--print`-only, so even a successful TTY-spoof
  would still degrade to TUI scraping for any automation-grade
  output.

We also considered keeping the existing Go CLI architecture and
swapping the auth path to a direct Anthropic API key or a third-party
provider (Bedrock / Vertex / Foundry). This works technically but
abandons subscription billing entirely and shifts paintress to a
per-token cost model with no upper bound, which is the opposite of
the design goal that motivated subscription onboarding.

The refs/issues/0027 plan synthesised these constraints into a single
direction: invert the LLM ownership.

## Decision

paintress relinquishes the LLM owner role. From this commit forward,
the architecture is:

1. **Human-initiated claude code interactive session is the LLM
   owner.** All inference happens inside the session's subscription
   quota. No production code path may invoke `claude -p`, import the
   Anthropic Agent SDK, read `ANTHROPIC_API_KEY`, or otherwise call
   the API outside the active session.
2. **paintress Go CLI exposes an MCP server (`paintress mcp`).** The
   server speaks JSON-RPC 2.0 over stdio and registers tools that
   wrap the existing data plane (event sourcing, gradient gauge,
   journal, D-Mail outbox/inbox, OTel instrumentation). The session
   loads the server via `--mcp-config`.
3. **A claude code skill drives the workflow.** The
   `/expedition-next` slash command (under `plugins/paintress/skills/`)
   is the only sanctioned entry point. Hooks may emit human-readable
   notices on stderr but must not auto-trigger LLM calls and must not
   surface inbox payloads on stdout (which the official hooks docs
   feed into the session's context).
4. **D-Mail cross-tool messaging is split by phase.** Emit uses an
   MCP tool, route stays in the phonewave daemon (which does not
   invoke LLMs and is unaffected by the credit pool change), and
   consume happens only via an explicit slash command - never via a
   SessionStart hook.
5. **A semgrep gate enforces the boundary.** The rule set in
   `.semgrep/jun15-no-headless-llm.yaml` blocks every executable
   path that could re-introduce `claude -p`, the Agent SDK,
   `ANTHROPIC_API_KEY`, or shell-wrapped variants thereof. `permanent`
   nosemgrep exemptions on this rule are not allowed in production
   paths; the only legitimate exclusions are test fixtures.

The Go CLI keeps its event sourcing, transactional outbox, OTel
spans, and domain language. What it loses is the right to spawn a
subprocess that calls the model.

## Enforcement inventory

### Entry points

- `cmd/paintress` Cobra commands - all subcommands that previously
  drove an expedition (notably `paintress run`).
- `internal/session/expedition.Run()` - the legacy LLM invocation
  loop that prior to this ADR built a prompt, exec'd `claude -p`,
  parsed stream-json, and managed mid-expedition watchers.
- Any future code that wants to call the model from outside an
  interactive session (Agent SDK, GitHub Actions, third-party SDK
  apps).

### Persistent data carried into the new path

- `internal/domain/dmail_envelope.go` (9-field DMailEnvelope) pins
  the cross-tool message schema (message_id, source_tool,
  target_tool, kind, body_path, created_at, seen_at, ack_at,
  idempotency_key).
- `paintress.next_issue` / `paintress.update_gradient` /
  `paintress.append_journal` MCP tools expose the existing journal,
  gradient gauge, and Linear-derived issue queue as MCP resources.
- OTel span attributes (`gen_ai.*`, `messaging.*`) continue to flow
  through the MCP server, so the trace topology that previously
  spanned `claude -p` invocations now spans MCP tool calls.

### Bypass candidates

- Direct `exec.Command("claude", ...)` from Go code (blocked by
  `jun15-no-claude-print-exec-go`).
- Shell wrappers (`bash -lc "claude -p ..."`, `sh -c`, `just`
  recipes, `scripts/*.sh`) - blocked by
  `jun15-no-claude-print-shell-wrapper` and the literal-scan rule.
- Anthropic SDK imports in Go, TypeScript, or Python - blocked by
  the SDK-import rules in the same yaml.
- `SessionStart` / `PreToolUse` hooks that stream inbox content on
  stdout - blocked by the documentation convention (`stderr only`)
  and the `type: prompt` hook prohibition.
- Future `--bare`-mode invocations of `claude` from outside the
  session - same shell-wrapper rule covers this.

### Tests proving coverage

- `internal/session/mcp_server_test.go` - four tests prove the
  `paintress mcp` stdio server advertises and dispatches the four
  Phase 1 tools (paintress.ping plus the three stubs).
- `internal/session/expedition_test.go::TestExpedition_Run_IsDeprecatedPostMCPPivot`
    - one test proves Expedition.Run() returns
  `ErrMCPPivotDeprecated` and produces no output.
- `internal/domain/dmail_envelope_test.go` - five tests cover the
  YAML schema, required-field validation, idempotency-key dedup,
  ack semantics, and the inbox/<id>.yaml + body.md file pair.
- `just semgrep` - 75 rules, 0 findings, including the five
  jun15-no-headless-llm gate rules with no production-path
  exclusions.

## Consequences

### Positive

- Subscription billing keeps paying for all paintress LLM use after
  2026-06-15. Credit pool consumption from paintress is zero by
  construction.
- The Acceptable Use Policy boundary is honoured: every model call
  is human-initiated inside an interactive session.
- The Go CLI's domain plane (event sourcing, SQLite outbox, OTel,
  D-Mail semantics) survives intact and is now exposed via a
  stable MCP contract that other tools can adopt.
- The semgrep gate makes the boundary mechanical; future
  contributors cannot silently re-introduce headless LLM calls.

### Negative

- `paintress run` and other autonomous CLI entry points return
  `ErrMCPPivotDeprecated`. Operators must launch a claude code
  session and run `/expedition-next` manually. Schedulers and CI
  jobs that wrapped `paintress run` no longer work without that
  human-in-the-loop step.
- Multi-tool parallel orchestration loses the easy concurrency
  story that came with independent Go processes. A single
  interactive session is the natural unit of work.
- Several test suites that exercised the legacy Run() body
  (mid-expedition D-Mail routing, worker isolation, stale flag
  clearing) were retired. Phase 2 must reintroduce equivalent
  coverage against the MCP server contract.

### Neutral

- Helper functions (BuildPrompt, loadInboxSection,
  loadContextSection, ReadContextFiles) are retained even though
  Run() no longer calls them, because the Phase 2 MCP server tools
  are expected to reuse them. Their `nolint` annotations carry
  explicit expiry dates.
- The dotfiles plugin sketches under `~/dotfiles/plugins/paintress/`
  remain as prototypes; the production target is the in-repo
  `plugins/paintress/` directory that ships alongside this Go CLI.

## References

- refs/issues/0027 - canonical plan including all four codex review
  rounds and the acceptance criteria checklist.
- ADR 0009 (event sourcing), ADR 0010 (event sourcing with
  projections), ADR 0011 (usecase / adapter dependency inversion)
    - the architectural layers this ADR preserves.
- ADR 0005 (three-way approval contract), ADR 0013 (preflight
  D-Mail triage) - the D-Mail invariants the new envelope schema
  must keep honouring once Phase 2 wiring lands.
- <https://code.claude.com/docs/en/headless> - 2026-06-15 credit
  pool change announcement and `--bare` mode documentation.
- <https://support.claude.com/en/articles/15036540> - per-plan
  credit allocation table.
