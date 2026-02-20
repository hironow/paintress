## paintress issues

List Linear issues

### Synopsis

List Linear issues assigned to the configured team and project.

Reads the Linear API key from the LINEAR_API_KEY environment variable
and the team/project from .expedition/config.yaml. Supports filtering
by issue state (e.g. todo, in-progress). Hyphens in state names are
converted to spaces automatically.

```
paintress issues <repo-path> [flags]
```

### Examples

```
  # List all issues
  paintress issues /path/to/repo

  # Filter by state
  paintress issues --state todo,in-progress /path/to/repo

  # JSON output for scripting
  paintress issues -o json /path/to/repo
```

### Options

```
  -h, --help           help for issues
      --state string   Comma-separated state filter (e.g. todo,in-progress)
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
