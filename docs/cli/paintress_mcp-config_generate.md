## paintress mcp-config generate

Generate .mcp.json and .claude/settings.json for subprocess isolation

### Synopsis

Generate .mcp.json and .claude/settings.json for Claude subprocess isolation.

.mcp.json controls which MCP servers are available:
  - wave mode (default): empty config (no MCP servers)
  - linear mode (--linear): includes Linear MCP server

.claude/settings.json disables all plugins for the subprocess.

Claude subprocess uses --strict-mcp-config to enforce the MCP allowlist.

```
paintress mcp-config generate [path] [flags]
```

### Options

```
      --force   Overwrite existing .mcp.json
  -h, --help    help for generate
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

* [paintress mcp-config](paintress_mcp-config.md)	 - Manage MCP configuration for Claude subprocess isolation

