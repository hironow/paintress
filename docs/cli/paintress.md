## paintress

Claude Code expedition orchestrator

### Synopsis

The Paintress — drives the Expedition loop for Claude Code.

### Options

```
  -c, --config string   Config file path
  -h, --help            help for paintress
  -l, --lang string     Output language: en, ja (default from config)
      --linear          Use Linear MCP for issue tracking (default: wave-centric mode)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -q, --quiet           Suppress all stderr output
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress archive-prune](paintress_archive-prune.md)	 - Prune old archived d-mails
* [paintress clean](paintress_clean.md)	 - Remove state directory (.expedition/)
* [paintress config](paintress_config.md)	 - View or update paintress project configuration
* [paintress dead-letters](paintress_dead-letters.md)	 - Manage dead-lettered outbox items
* [paintress doctor](paintress_doctor.md)	 - Run health checks
* [paintress init](paintress_init.md)	 - Initialize project configuration
* [paintress issues](paintress_issues.md)	 - List Linear issues via Claude MCP
* [paintress mcp-config](paintress_mcp-config.md)	 - Manage MCP configuration for Claude subprocess isolation
* [paintress rebuild](paintress_rebuild.md)	 - Rebuild projections from event store
* [paintress run](paintress_run.md)	 - Run the expedition loop
* [paintress sessions](paintress_sessions.md)	 - Manage AI coding sessions
* [paintress status](paintress_status.md)	 - Show paintress operational status
* [paintress update](paintress_update.md)	 - Self-update paintress to the latest release
* [paintress version](paintress_version.md)	 - Print version, commit, and build information

