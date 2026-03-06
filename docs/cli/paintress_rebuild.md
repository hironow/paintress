## paintress rebuild

Rebuild projections from event store

### Synopsis

Replays all events from .expedition/events/ to regenerate materialized projection state from scratch.

```
paintress rebuild [repo-path] [flags]
```

### Options

```
  -h, --help   help for rebuild
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator

