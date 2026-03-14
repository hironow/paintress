## paintress rebuild

Rebuild projections from event store

### Synopsis

Replays all events from .expedition/events/ to regenerate materialized projection state from scratch.

If path is omitted, the current working directory is used.

```
paintress rebuild [path] [flags]
```

### Examples

```
  # Rebuild projections for the current directory
  paintress rebuild

  # Rebuild projections for a specific project
  paintress rebuild /path/to/repo
```

### Options

```
  -h, --help   help for rebuild
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja (default from config)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator

