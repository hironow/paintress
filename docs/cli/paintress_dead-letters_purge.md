## paintress dead-letters purge

Purge dead-lettered outbox items

### Synopsis

Purge outbox items that have exceeded the maximum retry count (3+ failures).

By default runs in dry-run mode, showing the count of dead-lettered items.
Use --execute to perform actual deletion.

```
paintress dead-letters purge [path] [flags]
```

### Examples

```
  # Dry run: show dead-letter count (current directory)
  paintress dead-letters purge

  # Dry run: show count for a specific project
  paintress dead-letters purge /path/to/repo

  # Delete dead-lettered items
  paintress dead-letters purge --execute

  # Delete without confirmation prompt
  paintress dead-letters purge --execute --yes
```

### Options

```
  -x, --execute   Execute purge (default: dry-run)
  -h, --help      help for purge
  -y, --yes       Skip confirmation prompt
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

* [paintress dead-letters](paintress_dead-letters.md)	 - Manage dead-lettered outbox items

