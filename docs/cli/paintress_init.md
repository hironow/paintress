## paintress init

Initialize project configuration

### Synopsis

Initialize a .expedition/ directory in the target repository.

Creates config.yaml with Linear team key, project name, and
default expedition settings. This must be run once before
'paintress run' can operate on the repository.

```
paintress init <repo-path> [flags]
```

### Examples

```
  # Initialize a new project
  paintress init /path/to/repo

  # Initialize and then run
  paintress init /path/to/repo && paintress run /path/to/repo
```

### Options

```
  -h, --help   help for init
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
