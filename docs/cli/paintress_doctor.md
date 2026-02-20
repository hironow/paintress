## paintress doctor

Check external command availability

### Synopsis

Check that all external commands required by paintress are installed.

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and
docker. Reports version and path for each found command.

```
paintress doctor [flags]
```

### Examples

```
  # Check all dependencies
  paintress doctor

  # Machine-readable output
  paintress doctor -o json
```

### Options

```
  -h, --help   help for doctor
```

### Options inherited from parent commands

```
  -c, --config string   Path to config file
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator
