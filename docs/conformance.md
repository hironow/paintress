# What / Why / How Conformance

This is the single source of truth for paintress's purpose, design rationale, and implementation approach.
Referenced from [README.md](../README.md) and [docs/README.md](README.md).

| Aspect | Description |
|--------|-------------|
| **What** | Autonomous loop that sends AI agents to implement Linear issues until the backlog reaches zero |
| **Why** | Automate the full issue lifecycle: implement, test, PR, review, fix — with no human intervention |
| **How** | Fetch issue via Claude MCP → Claude Code implements in git branch → review gate → PR → next issue |
| **Input** | Linear issues via Claude MCP tools, repository source code |
| **Output** | Git branches, pull requests, D-Mail reports to downstream tools |
| **Telemetry** | OTel spans: `paintress.run`, `expedition`, `claude.invoke` (with `claude.model`, `claude.timeout_sec`, `gen_ai.*`) |
| **External Systems** | Linear (via Claude MCP), Claude Code subprocess, Git/GitHub, OTel exporter (Jaeger/Weave) |

## Cross-Tool Conformance

All 4 tools (phonewave, sightjack, paintress, amadeus) maintain a What/Why/How conformance table in `docs/conformance.md` with the same structure. This prevents expression drift across README files.
