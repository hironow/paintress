---
name: expedition-next
description: >-
  Slash command for the paintress expedition runner (jun15 MCP pivot).
  Triggers when the user types "/expedition-next", asks to "pick the
  next paintress issue", "run one paintress expedition via MCP",
  "次の expedition を実行して", or "test the paintress MCP server
  end-to-end". Drives the paintress MCP server's tools (next_issue /
  update_gradient / append_journal) from inside a human-initiated
  Claude Code interactive session so inference stays on the
  subscription quota rather than the Agent SDK credit pool that gates
  `claude -p` from 2026-06-15.
version: 0.2.0
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
this interactive Claude Code session's subscription quota.

## Execution principle: one invocation = one expedition

One `/expedition-next` run handles **exactly one issue → one branch →
one PR**, then stops and reports back to the human. Do not loop into
the next issue automatically — the human re-invokes the slash command
for each expedition. This keeps the feedback loop negative (stable,
human-paced) and prevents runaway sessions.

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
the continent (`.expedition/` journal + event store). The MCP server
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
     "journal_dir": "/path/to/project/.expedition/journal",
     "instruction": "Read the configured issue source, exclude completed_issue_ids, pick the highest-priority unstarted item. Persist completion via paintress.append_journal after the expedition."
   }
   ```

   If `initialized == false`, **abort**: the operator launched
   `paintress mcp` from outside a paintress-initialized project root.
   Ask them to relaunch `claude` from the project directory.

3. **Pick the next issue from the configured issue source (wave
   mode)**. The default issue source is the **specification D-Mails
   that sightjack produced and phonewave delivered into
   `.expedition/inbox/`** (`kind: specification`, YAML frontmatter +
   Markdown). Linear is NOT used (wave mode replaced it; shared ADR
   S0035).

   - `Glob` for `.expedition/inbox/*.md`, `Read` the frontmatter, and
     collect the issue ids the specs describe.
   - Exclude every id in `completed_issue_ids` from step 2.
   - Pick the highest-priority unstarted item; tie-break by oldest.
   - Reading inbox files is safe (phonewave delivers atomically via
     temp-file-rename); never write to `inbox/` or move its files.
   - If the inbox holds no unstarted spec, report "no work available"
     and stop — do not invent work.

4. **Implement the fix on a branch**. Read the spec body, plan the
   change, then:

   - create a working branch (e.g. `fix/...` or `feat/...`),
   - apply edits via Read / Edit / Write / Bash,
   - validate with the project's test command (configured in
     `.expedition/config.yaml`); the expedition is only "done" when
     verification passes,
   - commit in Conventional Commits form (structural and behavioral
     changes in separate commits), push, and open a PR via
     `gh pr create` with neutral wording.

   No `claude -p` invocations are allowed at any point.

5. **Update the gradient gauge**. Call
   `mcp__paintress__paintress_update_gradient` with `{"delta": <signed>}`
   — `+1` for success, `-1` for failure. The tool reads the current
   level from the event store, applies the delta, persists an
   `EventGradientChanged` event (`persistence: "event-store"`), and
   returns `current_level` + `new_level`.

6. **Append the journal entry**. Call
   `mcp__paintress__paintress_append_journal` with the expedition
   metadata (expedition number / issue_id / status / pr_url / etc.).
   The tool writes `journal/<NNN>.md` + the pr-index AND persists an
   `EventExpeditionCompleted` event
   (`persistence: "event-store+filesystem"`).

7. **Report**. End with: expedition number, issue id, PR URL,
   verification result, gradient change, and what the human should do
   next (review the PR / re-invoke for the next expedition).

## Failure paths

- **MCP tool error mid-run**: report the tool name and the error
  surface, stop. Do not retry more than once.
- **Verification failure you cannot fix within the spec's scope**:
  record the failure via `update_gradient` with `delta: -1`, leave the
  branch unpushed (or push as draft if partially valuable), report
  exactly what failed (command + output tail), stop.
- **Ambiguous spec**: ask the human instead of guessing — a
  specification D-Mail is a contract, not a suggestion.

## Re-run idempotency

Re-invoking `/expedition-next` after a partial run is safe:
`next_issue` recomputes `completed_issue_ids` from the event store, so
an issue is only excluded once `append_journal` has persisted it.
Before re-implementing, check whether the working branch from the
aborted run already exists (`git branch --list`) and resume it instead
of starting over. Never call `append_journal` twice for the same
expedition number.

## What this skill must NOT do

- Invoke `claude -p`, `claude --print`, the Anthropic Agent SDK, or
  any shell wrapper that does so (= billing boundary). The repo-wide
  semgrep gate (`.semgrep/jun15-no-headless-llm.yaml`) blocks these
  patterns in production code.
- Auto-trigger inference from a SessionStart hook or any other
  non-human-initiated path. The slash command typed by a human is
  the only valid entry to this workflow.
- Query Linear / linear-mcp for issues. Wave mode replaced Linear
  (shared ADR S0035); the issue source is the spec D-Mail inbox.
- Emit a report D-Mail by writing to `outbox/` directly. Direct writes
  bypass the transactional outbox (atomicity / idempotency / OTel
  audit). A `paintress.dmail` emission tool does not exist yet — that
  gap is tracked in refs issue 0031; until it lands, report D-Mail
  emission is out of scope for this skill (the journal + gradient
  events are the durable completion record).

## Done criteria

An `/expedition-next` run is complete when, in a real Claude Code
session with the paintress MCP server attached:

1. `ping` returns `pong` (handshake + tool dispatch verified).
2. `next_issue` returns `initialized: true` with the journal state.
3. One spec is implemented on a branch, verification passes, and a PR
   exists.
4. `update_gradient` returns `persistence: "event-store"` and
   `append_journal` returns `persistence: "event-store+filesystem"`,
   so the expedition is durably recorded.
5. The closing report (expedition / issue / PR / gradient / next step)
   is delivered to the human.

## Related

- Canonical plan: `http://localhost:8765/docs/archive/0027-jun15-mcp-pivot.html` (refs)
- refs restructure + skill review: `http://localhost:8765/docs/issues/0030-refs-attic-restructure.html`
- D-Mail emission tool gap: `http://localhost:8765/docs/issues/0031-mcp-tool-surface-gaps.html`
- Pattern reference:
    - paintress ADR 0017 (`docs/adr/0017-mcp-pivot.md`) — MCP pivot
    - paintress ADR 0018 (`docs/adr/0018-mcp-pivot-helper-level-stub.md`) — helper-level stub
- Wave mode: shared ADR S0035 (`docs/shared-adr/S0035-dmail-wave-field-extension.md`)
- Mechanical gate (semgrep rules): `.semgrep/jun15-no-headless-llm.yaml`
- D-Mail 9-field schema: `internal/domain/dmail_envelope.go`
