## paintress status

Show paintress operational status

### Synopsis

Display operational status including expedition history, success rate,
gradient level, and pending d-mail counts.

Output goes to stdout by default (human-readable text).
Use -o json for machine-readable JSON output to stdout.

```
paintress status [path] [flags]
```

### Examples

```
  # Show status for current directory
  paintress status

  # Show status for a specific project
  paintress status /path/to/repo

  # JSON output for scripting
  paintress status -o json /path/to/repo
```

### Options

```
  -h, --help   help for status
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja (default from config)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -q, --quiet           Suppress all stderr output
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator

