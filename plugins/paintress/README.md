# paintress claude code plugin (jun15 MCP pivot)

Skills and (later) agents that drive the paintress MCP server from a
human-initiated claude code interactive session, per refs/issues/0027
(jun15 MCP pivot Phase 1 MVP).

## Layout

```
plugins/paintress/
├── README.md                          # this file
└── skills/
    └── expedition-next/SKILL.md       # /expedition-next slash command
```

Subsequent commits on `feat/jun15-mcp-pivot` add:

- `agents/expedition.md` — long-running TDD loop (post-stub)
- `skills/check-inbox/SKILL.md` — explicit D-Mail consume entry point
- `hooks/` — non-LLM hooks only (e.g. stderr-only inbox count notice)

## Loading the plugin

```bash
claude \
  --plugin-dir ./plugins/paintress \
  --mcp-config '{"paintress":{"command":"paintress","args":["mcp"]}}'
```

The `--plugin-dir` flag registers the skills; the `--mcp-config` flag
attaches the paintress MCP server (`paintress mcp` subcommand) so the
skill's `mcp__paintress__*` tools resolve.

## Phase 1 MVP scope

Only `/expedition-next` is wired. The slash command calls the paintress
MCP server's stub tools (paintress.ping, paintress.next_issue,
paintress.update_gradient, paintress.append_journal) and surfaces the
stub contract to the human. Real domain wiring lands in subsequent
commits on `feat/jun15-mcp-pivot`.

## Why this lives in the paintress repo (not dotfiles)

The dotfiles `plugins/paintress/` directory is a prototype sketch that
predates the jun15 pivot. The version here is the production target:
versioned alongside the Go MCP server it drives, gated by the same
semgrep rules (refs 0027 §6), and pinned by ADR (post-pivot).
