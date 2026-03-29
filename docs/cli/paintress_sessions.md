## paintress sessions

Manage AI coding sessions

### Synopsis

Manage AI coding session records. Sessions are tracked in SQLite
and can be listed, filtered, and re-entered interactively.

### Examples

```
  paintress sessions list
  paintress sessions list --status completed --limit 5
  paintress sessions enter <session-record-id>
  paintress sessions enter --provider-id <claude-session-id>
```

### Options

```
  -h, --help   help for sessions
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja (default from config)
      --linear          Use Linear MCP for issue tracking (default: wave-centric mode)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -q, --quiet           Suppress all stderr output
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator
* [paintress sessions enter](paintress_sessions_enter.md)	 - Re-enter an AI coding session interactively
* [paintress sessions list](paintress_sessions_list.md)	 - List recorded coding sessions

