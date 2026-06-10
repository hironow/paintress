# paintress Claude Code integration

The `/expedition-next` entry skill moved into the embedded templates at
`internal/platform/templates/claude-skills/expedition-next/SKILL.md`
(single source of truth). `paintress init` materializes it into the
target project's `.claude/skills/` so a bare `claude` session
auto-discovers it, and `paintress mcp-config generate` upserts the
project-root `.mcp.json` (merge-aware) so the MCP server auto-attaches
(refs issue 0032, decision D5).

This directory is kept as a pointer; no plugin manifest machinery is
used.
