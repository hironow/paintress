## paintress mcp

Run paintress as an MCP server over stdio (expedition journal/gradient data plane)

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

Exposes paintress.ping, paintress.next_issue (reads journal +
pr-index to surface completed issue ids + next expedition number),
and paintress.update_gradient + paintress.append_journal (persist
gradient / expedition-completed events to the event store, with a
journal/ + pr-index filesystem write).

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

* [paintress](paintress.md)	 - Expedition journal/gradient MCP data plane

