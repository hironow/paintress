## paintress issues

List Linear issues via Claude MCP

### Synopsis

Query Linear issues via Claude MCP tools for the configured team and project.

Reads the team/project from .expedition/config.yaml. Supports filtering
by issue state (e.g. todo, in-progress). Hyphens in state names are
converted to spaces automatically.

If repo-path is omitted, the current working directory is used.

```
paintress issues [repo-path] [flags]
```

### Examples

```
  # List all issues (current directory)
  paintress issues

  # List all issues (explicit path)
  paintress issues /path/to/repo

  # Filter by state
  paintress issues --state todo,in-progress /path/to/repo

  # JSON output for scripting
  paintress issues -o json /path/to/repo
```

### Options

```
  -h, --help           help for issues
  -s, --state string   Comma-separated state filter (e.g. todo,in-progress)
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress](paintress.md)  - Claude Code expedition orchestrator
