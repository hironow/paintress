---
name: expedition-next
description: >-
  Phase 1 MVP slash command for the paintress jun15 MCP pivot
  (refs/issues/0027). Triggers when the user types "/expedition-next",
  asks to "pick the next paintress issue", "run one paintress expedition
  via MCP", or "test the paintress MCP server end-to-end". Drives the
  paintress MCP server's stub tools (next_issue / update_gradient /
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

# /expedition-next — paintress MCP pivot Phase 1 MVP

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

## Workflow

1. **Verify MCP wiring**. Call `mcp__paintress__paintress_ping`. The
   tool must return `pong`. If it errors, the MCP server is not
   attached — abort and ask the human to relaunch claude with
   `--mcp-config`.

2. **Fetch journal state from paintress**. Call
   `mcp__paintress__paintress_next_issue` with no arguments. Real impl
   (Phase 3) returns paintress's local journal state:

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

5. **Implement the fix**. Read the issue body, plan the
   change, and apply edits via Read / Edit / Write / Bash. Use the
   project's existing test command (configured in `continent-config.yaml`)
   to validate. No `claude -p` invocations are allowed at any point.

6. **(Post-stub) Update the gradient gauge**. Call
   `mcp__paintress__paintress_update_gradient` with `{"delta": <signed>}`
   to record success (+1) or failure (-1). The response is currently
   a stub that echoes the delta as `new_level`.

7. **(Post-stub) Append the journal entry**. Call
   `mcp__paintress__paintress_append_journal` with the expedition
   metadata (expedition number / issue_id / status / pr_url / etc.).
   Phase 1 stub echoes the entry without persisting; the real wiring
   commit hooks this into the existing JournalEntry event sourcing.

## What this skill must NOT do

- Invoke `claude -p`, `claude --print`, the Anthropic Agent SDK, or
  any shell wrapper that does so (= refs/issues/0027 §5 billing
  boundary). The repo-wide semgrep gate
  (`.semgrep/jun15-no-headless-llm.yaml`) blocks these patterns in
  production code.
- Auto-trigger inference from a SessionStart hook or any other
  non-human-initiated path. The slash command typed by a human is
  the only valid entry to this workflow.
- Emit a D-Mail by writing to `outbox/` directly. Use
  `mcp__paintress__paintress_emit_dmail` once it lands in a later
  commit; that tool encapsulates the transactional outbox + the
  9-field schema fixed in refs 0027 §8.

## Phase 1 MVP exit criteria

This skill is considered Phase 1 MVP complete when:

1. Calling `/expedition-next` in a real claude code session with the
   paintress MCP server attached returns the stub responses from
   steps 1-2 without error.
2. The synthetic D-Mail contract test (= subsequent commit) drives a
   fixture through `inbox/` and proves consume happens only when the
   human types `/expedition-next`, never from a hook.
3. `expedition.go`'s `claude -p` invocation is removed and the
   semgrep transitional exclude on `internal/session/expedition.go`
   is deleted (= the final commit on the `feat/jun15-mcp-pivot`
   branch flips the lint gate from advisory to enforced).

## Related

- Canonical plan: `refs/HTMLification/docs/issues/0027-jun15-mcp-pivot.html`
- Billing boundary table: refs 0027 §5
- Mechanical gate (semgrep rules): refs 0027 §6 + `.semgrep/jun15-no-headless-llm.yaml`
- D-Mail 9-field schema (Phase 1 固定): refs 0027 §8
