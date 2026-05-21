## paintress issues

DEPRECATED post jun15 MCP pivot (refs/issues/0027)

### Synopsis

DEPRECATED post jun15 MCP pivot.

The previous implementation invoked the Claude CLI subprocess to query
Linear issues via the Linear MCP tools. Post jun15 MCP pivot
(refs/issues/0027 + 0028 §4.2 residue cleanup), headless Claude
invocations are forbidden. Use claude code with the paintress MCP
server attached instead:

  claude --plugin-dir ./plugins/paintress \
         --mcp-config '{"paintress":{"command":"paintress","args":["mcp"]}}'

Then call the paintress.next_issue MCP tool from your session.

```
paintress issues [path] [flags]
```

### Options

```
  -h, --help           help for issues
  -s, --state string   (deprecated) state filter, no longer applied
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

