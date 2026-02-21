# CLAUDE.md

## Architecture Decisions

### CLI structure: `internal/cmd/` (not `cmd/`)

paintress uses `internal/cmd/` instead of the cobra-cli default `cmd/` directory.

Reason: paintress is a standalone CLI binary with no need to export command logic to other packages. Using `internal/` prevents external import of command definitions while keeping the `cmd/paintress/main.go` entry point minimal (~20 lines). This structure was agreed upon across all 4 tools in the MY-329 consensus (2026-02).

### cobra framework

CLI is built on `spf13/cobra` v1.10.2.

- `NewRootCommand()` is exported from `internal/cmd` for testability (`SetArgs`/`SetOut`) and future docgen
- `cobra.EnableTraverseRunHooks = true` for PersistentPreRunE propagation
- All commands use `RunE` (not `Run`) — error handling centralized in `main()`
- `--output` and `--lang` are PersistentFlags on root, inherited by all subcommands
- `--review-cmd` default is dynamically derived from `--base-branch` in `PreRunE`

## Build

```bash
just build    # builds with version from git tags
just install  # builds and installs to /usr/local/bin
```

Version is injected via `-ldflags "-X github.com/hironow/paintress/internal/cmd.Version=..."`.

## Testing

```bash
just test       # all tests, 300s timeout
just test-v     # verbose
just test-race  # with race detector
```

Container tests (`testcontainers-go`) require Docker and may take ~11s each. `testing.Short()` skips them.
