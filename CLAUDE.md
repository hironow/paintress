# paintress

## Workflow

- Do NOT use git worktrees (`EnterWorktree`, `isolation: "worktree"`). Work directly on the current branch.

## Repository Structure

- Entry: `cmd/paintress/main.go` (signal.NotifyContext + NeedsDefaultRun + ExitError)
- CLI: `internal/cmd/` (cobra v1.10.2, `NewRootCommand()` exported for testability)
- Library: root package `paintress` (expedition, dmail, gate, review, journal, inbox_watcher, etc.)
- OTel: `telemetry.go` (noop default + OTLP HTTP exporter)
- Docker: `docker/compose.yaml` + `docker/jaeger-v2-config.yaml` (Jaeger v2)
- Companions: `cmd/paintress-tg/`, `cmd/paintress-discord/`, `cmd/paintress-slack/` (notify/approve)
- Semgrep: `.semgrep/cobra.yaml` (canonical source is phonewave)

## CLI Design

- `cobra.EnableTraverseRunHooks = true` in `init()` (not constructor)
- All commands use `RunE` (not `Run`)
- `--output`, `--lang` are PersistentFlags on root
- Default subcommand: `paintress [flags] <repo>` → prepends `run` via `NeedsDefaultRun`

## Build & Test

```bash
just build       # build with version from git tags
just build-all   # build all binaries (main + companions)
just install     # build + install to /usr/local/bin
just install-all # install all binaries (main + companions)
just test        # all tests, 300s timeout
just test-race   # with race detector
just check       # fmt + vet + test
just semgrep     # cobra semgrep rules
just lint        # vet + markdown lint + gofmt check
```
