## paintress clean

Remove state directory (.expedition/)

### Synopsis

Delete the .expedition/ directory to reset to a clean state. Use 'paintress init' to reinitialize.

```
paintress clean [path] [flags]
```

### Examples

```
  # Clean the current directory
  paintress clean

  # Clean a specific project
  paintress clean /path/to/repo

  # Skip confirmation prompt
  paintress clean --yes
```

### Options

```
  -h, --help   help for clean
      --yes    Skip confirmation prompt
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

