## paintress archive-prune

Prune old archived d-mails

### Synopsis

Prune archived d-mail files older than a specified number of days.

By default runs in dry-run mode, listing candidates without deleting.
Use --execute to perform actual deletion. The archive/ directory is
git-tracked, so deletions should be reviewed and committed.

```
paintress archive-prune [path] [flags]
```

### Examples

```
  # Dry run: list files older than 30 days (current directory)
  paintress archive-prune

  # Dry run: list files for a specific project
  paintress archive-prune /path/to/repo

  # Delete files older than 14 days
  paintress archive-prune --days 14 --execute /path/to/repo

  # JSON output for scripting
  paintress archive-prune -o json /path/to/repo

  # Rebuild archive index from existing files
  paintress archive-prune --rebuild-index
```

### Options

```
  -d, --days int        Retention days (default 30)
  -n, --dry-run         Dry-run mode (default behavior, explicit for scripting)
  -x, --execute         Execute pruning (default: dry-run)
  -h, --help            help for archive-prune
      --rebuild-index   Rebuild archive index from existing files
  -y, --yes             Skip confirmation prompt
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
