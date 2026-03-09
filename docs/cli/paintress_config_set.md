## paintress config set

Update a project configuration value

### Synopsis

Update a configuration value in .expedition/config.yaml.

Supported keys:
  tracker.team     Linear team key (e.g. MY)
  tracker.project  Linear project name

```
paintress config set <key> <value> [path] [flags]
```

### Examples

```
  paintress config set tracker.team MY /path/to/repo
  paintress config set tracker.project "My Project" /path/to/repo
```

### Options

```
  -h, --help   help for set
```

### Options inherited from parent commands

```
  -l, --lang string     Output language: en, ja, fr (default "en")
      --no-color        Disable colored output (respects NO_COLOR env)
  -o, --output string   Output format: text, json (default "text")
  -v, --verbose         Enable verbose output
```

### SEE ALSO

* [paintress config](paintress_config.md)  - View or update paintress project configuration
