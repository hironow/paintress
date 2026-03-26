## paintress config set

Update a project configuration value

### Synopsis

Update a configuration value in .expedition/config.yaml.

Supported keys:
  tracker.team     Linear team key (e.g. MY)
  tracker.project  Linear project name
  tracker.cycle    Linear cycle name
  lang             Language (ja or en)
  max_expeditions  Maximum number of expeditions
  timeout_sec      Timeout per expedition in seconds
  model            Model(s) comma-separated (e.g. opus,sonnet)
  base_branch      Base branch (e.g. main)
  claude_cmd       Claude Code CLI command name
  dev_cmd          Dev server command
  dev_dir          Dev server working directory
  dev_url          Dev server URL
  review_cmd       Code review command after PR creation
  workers          Number of worktrees in pool (0 = direct)
  setup_cmd        Command to run after worktree creation
  no_dev           Skip dev server startup (true/false)
  notify_cmd       Notification command
  approve_cmd      Approval command
  auto_approve     Skip approval gate (true/false)
  max_retries      Maximum retry attempts per issue set

```
paintress config set <key> <value> [path] [flags]
```

### Examples

```
  paintress config set tracker.team MY /path/to/repo
  paintress config set model opus,sonnet /path/to/repo
  paintress config set workers 3
  paintress config set lang en
```

### Options

```
  -h, --help   help for set
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

* [paintress config](paintress_config.md)	 - View or update paintress project configuration

