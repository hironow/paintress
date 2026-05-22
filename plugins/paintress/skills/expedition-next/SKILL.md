---
name: expedition-next
description: >-
  Slash command for the paintress expedition runner (refs/issues/0027
  jun15 MCP pivot). Triggers when the user types "/expedition-next",
  asks to "pick the next paintress issue", "run one paintress expedition
  via MCP", or "test the paintress MCP server end-to-end". Drives the
  paintress MCP server's tools (next_issue / update_gradient /
  append_journal) from inside a human-initiated claude code interactive
  session so inference stays on the subscription quota rather than the
  Agent SDK credit pool that gates `claude -p` from 2026-06-15.
version: 0.1.0
argument-hint: "(none) - reads next issue from paintress MCP and runs one expedition"
allowed-tools:
  - Read
  - Edit
  - Write
  - Bash
  - Grep
  - Glob
  - Agent
  - mcp__paintress__paintress_ping
  - mcp__paintress__paintress_next_issue
  - mcp__paintress__paintress_update_gradient
  - mcp__paintress__paintress_append_journal
---

# /expedition-next — paintress expedition runner

Human-initiated entry point. Drives the paintress MCP server's tools
without ever invoking `claude -p`, so all inference happens inside
this interactive claude code session's subscription quota.

## Prerequisites

The session was launched with the paintress MCP server attached:

```bash
claude --mcp-config '{"paintress":{"command":"paintress","args":["mcp"]}}'
```

If `paintress mcp` is not on PATH, build it first:

```bash
cd path/to/paintress && go build -o ./dist/paintress ./cmd/paintress
```

`paintress mcp` must be started from the project root so it can resolve
the continent (`.paintress/` journals + event store). The MCP server
answers the `initialize` handshake, then exposes ping / next_issue /
update_gradient / append_journal.

## Workflow

1. **Verify MCP wiring**. Call `mcp__paintress__paintress_ping`. The
   tool must return `pong`. If it errors, the MCP server is not
   attached — abort and ask the human to relaunch claude with
   `--mcp-config`.

2. **Fetch journal state from paintress**. Call
   `mcp__paintress__paintress_next_issue` with no arguments. It returns
   paintress's local journal state from the event store:

   ```json
   {
     "initialized": true,
     "continent": "/path/to/project",
     "next_expedition_number": 5,
     "completed_issue_ids": ["X-1", "X-2", "X-3", "X-4"],
     "last_pr": {"expedition": 4, "issue_id": "X-4", "pr_url": "https://..."},
     "journal_dir": "/path/to/project/.paintress/journals",
     "instruction": "Query linear-mcp for unstarted issues, exclude completed_issue_ids, pick highest priority. Persist completion via paintress.append_journal after the expedition."
   }
   ```

   If `initialized == false`, **abort**: the operator launched
   `paintress mcp` from outside a paintress-initialized project root.
   Ask them to relaunch `claude` from the project directory.

   paintress does NOT query Linear itself (= that would re-introduce
   claude-driven inference). The session is responsible for fetching
   raw issues via `linear-mcp` and excluding `completed_issue_ids`.

3. **Query linear-mcp for the next unstarted issue**. With the
   completed_issue_ids from step 2, call your attached linear-mcp
   tool (e.g. `mcp__linear-mcp__list_issues`) and filter out those
   ids. Pick the highest-priority unstarted issue. If multiple have
   the same priority, prefer the oldest.

4. **Implement the fix**. Read the issue body, plan the change, and
   apply edits via Read / Edit / Write / Bash. Use the project's
   existing test command (configured in `continent-config.yaml`) to
   validate. No `claude -p` invocations are allowed at any point.

5. **Update the gradient gauge**. Call
   `mcp__paintress__paintress_update_gradient` with `{"delta": <signed>}`
   to record success (+1) or failure (-1). The tool reads the current
   level from the event store, applies the delta, and persists an
   `EventGradientChanged` event (`persistence: "event-store"`),
   returning `current_level` + `new_level`.

6. **Append the journal entry**. Call
   `mcp__paintress__paintress_append_journal` with the expedition
   metadata (expedition number / issue_id / status / pr_url / etc.).
   The tool writes `journal/<NNN>.md` + the pr-index AND persists an
   `EventExpeditionCompleted` event
   (`persistence: "event-store+filesystem"`).

## What this skill must NOT do

- Invoke `claude -p`, `claude --print`, the Anthropic Agent SDK, or
  any shell wrapper that does so (= refs/issues/0027 §5 billing
  boundary). The repo-wide semgrep gate
  (`.semgrep/jun15-no-headless-llm.yaml`) blocks these patterns in
  production code.
- Auto-trigger inference from a SessionStart hook or any other
  non-human-initiated path. The slash command typed by a human is
  the only valid entry to this workflow.
- Emit a D-Mail by writing to `outbox/` directly. D-Mail emission is
  not exposed as an MCP tool in this skill's tool set; the gradient +
  journal tools above are the canonical persistence paths. The D-Mail
  9-field schema is fixed in refs 0027 §8.

## Done criteria

An `/expedition-next` run is complete when, in a real claude code
session with the paintress MCP server attached:

1. `ping` returns `pong` (handshake + tool dispatch verified).
2. `next_issue` returns `initialized: true` with the journal state.
3. The fix is implemented and validated against the project test command.
4. `update_gradient` returns `persistence: "event-store"` and
   `append_journal` returns `persistence: "event-store+filesystem"`,
   so the expedition is durably recorded.

## Related

- Canonical plan: `refs/HTMLification/docs/archive/0027-jun15-mcp-pivot.html`
- Pattern reference:
  - paintress ADR 0017 (`~/tap/paintress/docs/adr/0017-mcp-pivot.md`) — MCP pivot
  - paintress ADR 0018 (`~/tap/paintress/docs/adr/0018-mcp-pivot-helper-level-stub.md`) — helper-level stub
- Billing boundary table: refs 0027 §5
- Mechanical gate (semgrep rules): refs 0027 §6 + `.semgrep/jun15-no-headless-llm.yaml`
- D-Mail 9-field schema: refs 0027 §8
