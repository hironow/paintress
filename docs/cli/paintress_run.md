## paintress run

Deprecated (jun15 MCP pivot): use claude code + /expedition-next skill

### Synopsis

Deprecated by the jun15 MCP pivot (2026-06-15 credit-pool split).

The autonomous expedition loop no longer drives a headless 'claude -p'
invocation. Run expeditions from a claude code interactive session via
the /expedition-next skill, which drives paintress's MCP tools
(next_issue / update_gradient / append_journal). Start the data-plane
server with 'paintress mcp'.

```
paintress run [path] [flags]
```

### Options

```
      --approve-cmd string      Approval command ({message} placeholder, exit 0 = approve)
      --auto-approve            Skip approval gate for HIGH severity D-Mail
  -b, --base-branch string      Base branch (default "main")
      --claude-cmd string       Claude Code CLI command name (default "claude")
      --dev-cmd string          Dev server command (default "npm run dev")
      --dev-dir string          Dev server working directory (defaults to repo path)
      --dev-url string          Dev server URL (default "http://localhost:3000")
  -n, --dry-run                 Generate prompts only
  -h, --help                    help for run
      --idle-timeout duration   D-Mail waiting phase timeout (0 = 24h safety cap, negative = disable waiting) (default 30m0s)
      --max-expeditions int     Maximum number of expeditions (default 50)
  -m, --model string            Model(s) comma-separated for reserve: opus,sonnet,haiku (default "opus")
      --no-dev                  Skip dev server startup
      --notify-cmd string       Notification command ({title}, {message} placeholders)
      --review-cmd string       Code review command after PR creation
      --setup-cmd string        Command to run after worktree creation (e.g. 'bun install')
  -t, --timeout int             Timeout per expedition in seconds (default: 33min) (default 1980)
  -w, --workers int             Number of worktrees in pool (0 = direct execution) (default 1)
```

### Options inherited from parent commands

```
  -c, --config string   Config file path
  -l, --lang string     Output language: en, ja (default from config)
      --linear          Use Linear MCP for issue tracking (default: wave-centric mode)
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -q, --quiet           Suppress all stderr output
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator

