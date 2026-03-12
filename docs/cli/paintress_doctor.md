## paintress doctor

Run health checks

### Synopsis

Check environment health and tool availability.

Each check reports one of four statuses:
- **OK** — Check passed
- **FAIL** — Check failed (exit code 1)
- **SKIP** — Check not applicable
- **WARN** — Advisory warning (exit code 0, not a failure)

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and docker.
When repo-path is provided (or defaults to current directory), also checks
.expedition/ structure, skills, config, and computes success rate metrics.

The context-budget check estimates total token usage and reports WARN when
the threshold (20,000 tokens) is exceeded. Per-category breakdown shows
token counts for tools, skills, plugins, mcp, and hooks, with the heaviest
category marked. Category-specific hints point to `.claude/settings.json`
for optimization.

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

