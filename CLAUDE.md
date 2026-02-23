# paintress

## Workflow

- Do NOT use git worktrees (`EnterWorktree`, `isolation: "worktree"`). Work directly on the current branch.

## Repository Structure

- Entry: `cmd/paintress/main.go` (signal.NotifyContext + NeedsDefaultRun + ExitError)
- CLI: `internal/cmd/` (cobra v1.10.2, `NewRootCommand()` exported for testability)
- Library: root package `paintress` (expedition, dmail, gate, review, journal, inbox_watcher, etc.)
- OTel: `telemetry.go` (noop default + OTLP HTTP exporter, shutdown via defer in run.go)
- Docker: `docker/compose.yaml` + `docker/jaeger-v2-config.yaml` (Jaeger v2)
- Companions: `cmd/paintress-tg/`, `cmd/paintress-discord/`, `cmd/paintress-slack/` (notify/approve)
- Semgrep: `.semgrep/cobra.yaml` (canonical source is phonewave)
- Release: `.goreleaser.yaml`

## CLI Design

- `cobra.EnableTraverseRunHooks = true` in `init()` (not constructor)
- All commands use `RunE` (not `Run`)
- `--output`, `--lang`, `--verbose` are PersistentFlags on root
- Default subcommand: `paintress [flags] <repo>` → prepends `run` via `NeedsDefaultRun`
- OTel tracer shutdown: `defer` with 5s timeout from `context.Background()` in run.go (no OnFinalize)
- `run` subcommand: `--timeout`, `--model`, `--base-branch`, `--workers`, `--notify-cmd`, `--approve-cmd`, `--auto-approve`, `--dry-run` local flags
- Interactive input: `StdinApprover` (bufio.Scanner in goroutine + channel + ctx.Done()) for gate approval
- `PAINTRESS_QUIET` env var: enables quiet logger mode
- Companion CLIs: shared pattern (notify/approve/doctor subcommands, `--timeout` persistent flag, local `exitError`)

## Test Layout

- Unit tests: `*_test.go` colocated with source (Go convention)
  - All tests use in-package testing (`package paintress`, `package cmd`, `package main`)
  - No external test packages — tests access unexported internals directly
- Container tests: `worktree_test.go` (testcontainers-go with `alpine/git:latest`, ~11s per test)
  - Entrypoint: SIGTERM-aware `trap 'exit 0' TERM; while :; do sleep 1; done` (NOT `sleep infinity`)
  - Skipped with `testing.Short()`
- Host integration tests: `review_loop_test.go` (shell scripts + local git, ~13s total)
- Race tests: `race_test.go` (dedicated concurrency tests, run with `just test-race`)
- CLI integration: `internal/cmd/*_test.go` (cobra command testing)
- Companion tests: `cmd/paintress-{tg,discord,slack}/*_test.go` (mockBot pattern)
- Test helpers: `test_helpers_test.go` (setupGitRepoWithBranch, gitIsolatedEnv)
- Helper process: `GO_TEST_HELPER_PROCESS=1` + `GO_TEST_HELPER_OUTPUT` + `GO_TEST_HELPER_EXIT_CODE` for exec.Cmd testing

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
