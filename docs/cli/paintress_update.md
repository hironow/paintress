## paintress update

Self-update paintress to the latest release

### Synopsis

Self-update paintress to the latest GitHub release.

Downloads the latest release, verifies the checksum, and replaces
the current binary. Use --check to only check for updates without
installing.

```
paintress update [flags]
```

### Examples

```
  # Check for updates
  paintress update --check

  # Update to the latest version
  paintress update
```

### Options

```
      --check   Check for updates without installing
  -h, --help    help for update
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator
