## paintress version

Show version information

### Synopsis

Print version, commit hash, build date, and Go version.

By default outputs a human-readable single line. Use --json
for structured output suitable for scripts and CI.

```
paintress version [flags]
```

### Examples

```
  # Print version info
  paintress version

  # JSON output for scripts
  paintress version --json
```

### Options

```
  -h, --help   help for version
      --json   Output version info as JSON
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator
