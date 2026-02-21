## paintress archive-prune

Prune old archived d-mails

### Synopsis

Prune archived d-mail files older than a specified number of days.

By default runs in dry-run mode, listing candidates without deleting.
Use --execute to perform actual deletion. The archive/ directory is
git-tracked, so deletions should be reviewed and committed.

```
paintress archive-prune <repo-path> [flags]
```

### Examples

```
  # Dry run: list files older than 30 days
  paintress archive-prune /path/to/repo

  # Delete files older than 14 days
  paintress archive-prune --days 14 --execute /path/to/repo

  # JSON output for scripting
  paintress archive-prune -o json /path/to/repo
```

### Options

```
      --days int   Number of days threshold (default 30)
      --execute    Execute deletion (dry-run by default)
  -h, --help       help for archive-prune
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)	 - Claude Code expedition orchestrator
