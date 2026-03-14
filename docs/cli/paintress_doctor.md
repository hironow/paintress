## paintress doctor

Run health checks

### Synopsis

Check environment health and tool availability.

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and docker.
When path is provided (or defaults to current directory), also checks
.expedition/ structure, skills, config, and computes success rate metrics.
Each check reports one of four statuses: OK (passed), FAIL (exit 1),
SKIP (dependency missing), WARN (advisory, exit 0).

The context-budget check estimates token consumption per category
(tools, skills, plugins, mcp, hooks) and marks the heaviest.
When the threshold (20,000 tokens) is exceeded, a category-specific
hint recommends adjusting .claude/settings.json.

Use --repair to auto-fix repairable issues (stale PID, missing SKILL.md, etc.).

```
paintress doctor [path] [flags]
```

### Examples

```
  # Check current directory
  paintress doctor

  # Check a specific project directory
  paintress doctor /path/to/project

  # Machine-readable output
  paintress doctor -o json

  # Auto-fix repairable issues
  paintress doctor --repair
```

### Options

```
  -h, --help     help for doctor
      --repair   Auto-fix repairable issues
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja (default from config)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)  - Claude Code expedition orchestrator
