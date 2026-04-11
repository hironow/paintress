## paintress mcp-config

Manage MCP configuration for Claude subprocess isolation

### Synopsis

Manage the .mcp.json file that controls which MCP servers
are available to Claude subprocess invocations.

Use 'generate' to create the initial config, then edit it to add or remove
MCP servers as needed. Claude subprocess uses --strict-mcp-config to enforce
this allowlist when the file exists.

### Examples

```
  paintress mcp-config generate
  paintress mcp-config generate --linear
  paintress mcp-config generate --force
```

### Options

```
  -h, --help   help for mcp-config
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
* [paintress mcp-config generate](paintress_mcp-config_generate.md)	 - Generate .mcp.json and .claude/settings.json for subprocess isolation

