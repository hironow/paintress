## paintress config

View or update paintress project configuration

### Synopsis

View or update the .expedition/config.yaml project configuration file.

### Examples

```
  paintress config show /path/to/repo
  paintress config set tracker.team MY /path/to/repo
```

### Options

```
  -h, --help   help for config
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
* [paintress config set](paintress_config_set.md)	 - Update a project configuration value
* [paintress config show](paintress_config_show.md)	 - Display project configuration

