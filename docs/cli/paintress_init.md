## paintress init

Initialize project configuration

### Synopsis

Initialize a .expedition/ directory in the target repository.

Use --team and --project flags for non-interactive mode, or omit
flags for interactive prompts. This must be run once before
'paintress run' can operate on the repository.

```
paintress init <repo-path> [flags]
```

### Examples

```
  # Non-interactive with flags
  paintress init --team MY --project Hades /path/to/repo

  # Defaults only (no prompts)
  paintress init /path/to/repo
```

### Options

```
  -h, --help             help for init
      --project string   Linear project name
      --team string      Linear team key (e.g. MY)
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)  - Claude Code expedition orchestrator
