## paintress mcp

Run paintress as an MCP server over stdio (refs/issues/0027 Phase 1 MVP)

### Synopsis

Start a Model Context Protocol server reading JSON-RPC 2.0
messages on stdin and writing responses on stdout.

Designed for embedding in a claude code interactive session via
--mcp-config so inference stays on the session's subscription quota
rather than crossing into the Agent SDK credit pool that gates
'claude -p' from 2026-06-15.

The continent (= project root) is resolved from the current working
directory. paintress.next_issue reads pr-index.jsonl + journal/ under
this directory to surface completed issue ids + next expedition
number. The claude code session itself queries linear-mcp for raw
issue data and uses paintress.next_issue's completed_issue_ids to
exclude already-done work.

Phase 1 MVP scope (Phase 3 real impl): paintress.ping + real
paintress.next_issue + 2 remaining stubs (paintress.update_gradient,
paintress.append_journal). Real wiring for the 2 stubs ships in
subsequent commits.

```
paintress mcp [flags]
```

### Options

```
  -h, --help   help for mcp
```

### Options inherited from parent commands

```
  -c, --config string   Config file path
  -l, --lang string     Output language: en, ja (default from config)
      --linear          Use Linear MCP for issue tracking (default: wave-centric mode)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -q, --quiet           Suppress all stderr output
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator

