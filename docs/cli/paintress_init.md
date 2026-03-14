## paintress init

Initialize project configuration

### Synopsis

Initialize a .expedition/ directory in the target repository.

If path is omitted, the current working directory is used.
Use --team and --project flags for non-interactive mode, or omit
flags for interactive prompts. This must be run once before
'paintress run' can operate on the repository.

```
paintress init [path] [flags]
```

### Examples

```
  # Initialize current directory
  paintress init

  # Non-interactive with flags
  paintress init --team MY --project Hades /path/to/repo

  # Re-initialize (overwrite config, keep state)
  paintress init --force --team MY --project Hades /path/to/repo
```

### Options

```
      --force                 Overwrite existing config (preserves state directories)
  -h, --help                  help for init
      --otel-backend string   OTel backend: jaeger, weave
      --otel-entity string    Weave entity/team (required for weave)
      --otel-project string   Weave project (required for weave)
      --project string        Linear project name
      --team string           Linear team key (e.g. MY)
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

