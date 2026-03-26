## paintress version

Print version, commit, and build information

### Synopsis

Print version, commit hash, build date, Go version, and OS/arch.

By default outputs a human-readable single line. Use --json
for structured output suitable for scripts and CI.

```
paintress version [flags]
```

### Examples

```
  paintress version
  paintress version -j
```

### Options

```
  -h, --help   help for version
  -j, --json   Output version info as JSON
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja (default from config)
      --linear          Use Linear MCP for issue tracking (default: wave-centric mode)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -q, --quiet           Suppress all stderr output
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)  - Claude Code expedition orchestrator
