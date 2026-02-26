# paintress

## Workflow

- Do NOT use git worktrees (`EnterWorktree`, `isolation: "worktree"`). Work directly on the current branch.

## Repository Structure (ADR 0013/0014: 3-Layer Separation)

Dependency direction: `internal/cmd` → `internal/session` → `internal/eventsource` → `paintress` (root)

### Root package `paintress` — types, constants, pure functions, go:embed
- `paintress.go` — Paintress type, core types, pure methods
- `expedition.go` — Expedition types, go:embed templates, pure prompt building
- `dmail.go` — DMail types, ParseDMail, MarshalDMail, ValidateDMail (pure)
- `config.go` — Config type, validation
- `project_config.go` — ProjectConfig type
- `flag.go` — ExpeditionFlag type, FlagPath
- `journal.go` — JournalEntry type
- `lumina.go` — Lumina type, FormatLuminaForPrompt (pure)
- `issues.go` — Issue type
- `gradient.go` — GradientGauge type, pure methods
- `reserve.go` — ReserveParty type, pure methods
- `report.go` — Report types
- `approve.go` — Approver interface
- `notify.go` — Notifier interface
- `doctor.go` — DoctorCheckResult types
- `archive_prune.go` — prune types
- `lang.go` — language constants
- `logger.go` — structured logger (root infrastructure per S0005)
- `telemetry.go` — Tracer (noop default, root infrastructure per S0005)
- `event.go` — Event envelope, EventType constants, EventStore interface, ValidateEvent, NewEvent

### `internal/eventsource/` — event store infrastructure
- `store_file.go` — FileEventStore (JSONL append-only, implements EventStore)
- `lifecycle.go` — FindExpiredEventFiles, PruneEventFiles

### `internal/session/` — all filesystem, network, subprocess I/O
- `paintress.go` — Paintress orchestrator (Run, main loop)
- `expedition.go` — expedition execution (subprocess, file I/O)
- `dmail.go` — D-Mail file I/O (archive, inbox, outbox)
- `config.go` — LoadConfig, SaveConfig
- `project_config.go` — LoadProjectConfig, SaveProjectConfig
- `flag.go` — ReadFlag, WriteFlag
- `flag_watcher.go` — FlagWatcher (filesystem polling)
- `inbox_watcher.go` — InboxWatcher (filesystem polling)
- `journal.go` — WriteJournal, ListJournalFiles
- `lumina.go` — ScanJournalsForLumina
- `issues.go` — FetchIssues (HTTP)
- `review.go` — RunReview (subprocess)
- `approve.go` — StdinApprover, CmdApprover, AutoApprover
- `notify.go` — CmdNotifier, NullNotifier
- `doctor.go` — RunDoctor, health check functions
- `archive_prune.go` — archive file discovery/deletion
- `init.go` — InitGateDir
- `worktree.go` — git worktree operations
- `devserver.go` — dev server management

### `internal/cmd/` — cobra CLI commands
- `root.go` — NewRootCommand, PersistentFlags
- `run.go` — run subcommand (main expedition)
- `telemetry.go` — initTracer (OTLP HTTP exporter setup, shutdown via cobra.OnFinalize)
- `init.go`, `doctor.go`, `issues.go`, `archive_prune.go`, `update.go`, `version.go`
- `default_run.go` — NeedsDefaultRun logic
- `errors.go` — ExitError handling

### Other
- Entry: `cmd/paintress/main.go` (signal.NotifyContext + NeedsDefaultRun + ExitError)
- Companions: `cmd/paintress-tg/`, `cmd/paintress-discord/`, `cmd/paintress-slack/` (notify/approve)
- Docker: `docker/compose.yaml` + `docker/jaeger-v2-config.yaml` (Jaeger v2)
- Semgrep: `.semgrep/cobra.yaml` (canonical source is phonewave)
- Release: `.goreleaser.yaml`

## CLI Design

- `cobra.EnableTraverseRunHooks = true` in `init()` (not constructor)
- All commands use `RunE` (not `Run`)
- `--output`, `--lang`, `--verbose` are PersistentFlags on root
- Default subcommand: `paintress [flags] <repo>` → prepends `run` via `NeedsDefaultRun`
- OTel tracer shutdown: PersistentPreRunE + `cobra.OnFinalize` + `sync.Once` (consistent across all 4 tools)
- `run` subcommand: `--timeout`, `--model`, `--base-branch`, `--workers`, `--notify-cmd`, `--approve-cmd`, `--auto-approve`, `--dry-run` local flags
- Interactive input: `StdinApprover` (bufio.Scanner in goroutine + channel + ctx.Done()) for gate approval
- `PAINTRESS_QUIET` env var: enables quiet logger mode
- Companion CLIs: shared pattern (notify/approve/doctor subcommands, `--timeout` persistent flag, local `exitError`)

## Test Layout

- Root tests: `*_test.go` colocated (pure function tests only, `package paintress`)
- Session tests: `internal/session/*_test.go` (I/O tests, `package session`)
- CLI tests: `internal/cmd/*_test.go` (cobra command testing, `package cmd`)
- Companion tests: `cmd/paintress-{tg,discord,slack}/*_test.go` (mockBot pattern)
- Container tests: `internal/session/worktree_test.go` (testcontainers-go, skipped with `testing.Short()`)
- Race tests: `internal/session/race_test.go` (dedicated concurrency tests, run with `just test-race`)
- Test helpers: `internal/session/test_helpers_test.go` (setupGitRepoWithBranch, gitIsolatedEnv)
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
