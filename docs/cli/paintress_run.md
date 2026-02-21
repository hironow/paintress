## paintress run

Run the expedition loop

### Synopsis

Run the expedition loop against a target repository.

Each expedition picks a Linear issue, creates a worktree branch,
invokes Claude Code to implement the change, opens a pull request,
and optionally runs a review cycle. The loop continues until
max-expeditions is reached or the issue queue is empty.

```
paintress run <repo-path> [flags]
```

### Examples

```
  # Run with defaults (opus model, 50 expeditions, 33min timeout)
  paintress run /path/to/repo

  # Run with sonnet fallback and 3 parallel workers
  paintress run --model opus,sonnet --workers 3 /path/to/repo

  # Dry run (generate prompts only, no Claude invocation)
  paintress run --dry-run /path/to/repo

  # Skip dev server and use custom review command
  paintress run --no-dev --review-cmd "pnpm lint" /path/to/repo
```

### Options

```
  -b, --base-branch string    Base branch (default "main")
      --claude-cmd string     Claude Code CLI command name (default "claude")
      --dev-cmd string        Dev server command (default "npm run dev")
      --dev-dir string        Dev server working directory (defaults to repo path)
      --dev-url string        Dev server URL (default "http://localhost:3000")
  -n, --dry-run               Generate prompts only
  -h, --help                  help for run
      --max-expeditions int   Maximum number of expeditions (default 50)
  -m, --model string          Model(s) comma-separated for reserve: opus,sonnet,haiku (default "opus")
      --no-dev                Skip dev server startup
      --review-cmd string     Code review command after PR creation
      --setup-cmd string      Command to run after worktree creation (e.g. 'bun install')
  -t, --timeout int           Timeout per expedition in seconds (default: 33min) (default 1980)
  -w, --workers int           Number of worktrees in pool (0 = direct execution) (default 1)
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator
