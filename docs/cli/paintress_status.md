## paintress status

Show paintress operational status

### Synopsis

Display operational status including expedition history, success rate,
gradient level, and pending d-mail counts.

Output goes to stdout by default (human-readable text).
Use -o json for machine-readable JSON output to stdout.

```
paintress status [repo-path] [flags]
```

### Examples

```
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
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator

