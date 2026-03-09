## paintress doctor

Run health checks

### Synopsis

Check environment health and tool availability.

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and docker.
When repo-path is provided (or defaults to current directory), also checks
.expedition/ structure, skills, config, and computes success rate metrics.

```
paintress doctor [repo-path] [flags]
```

### Examples

```
  # Check current directory
  paintress doctor

  # Check a specific project directory
  paintress doctor /path/to/project

  # Machine-readable output
  paintress doctor -o json
```

### Options

```
  -h, --help   help for doctor
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

