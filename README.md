# Paintress

**An autonomous expedition loop that picks Linear issues, implements code changes, opens PRs, and iterates through review cycles until the backlog is drained.**

Paintress uses [Claude Code](https://docs.anthropic.com/en/docs/claude-code) to automatically process Linear issues — implementing code, running tests, creating PRs, running code reviews, verifying UI, and fixing bugs — with no human intervention, until every issue is done. In Swarm Mode (`--workers N`), multiple expeditions run in parallel using git worktrees for isolation.

```bash
paintress --model opus,sonnet ./your-repo
```

This single command makes Paintress repeat the following cycle:

1. Fetch an unfinished issue from Linear
2. Analyze it and determine the mission type: implement / verify / fix
3. Claude Code creates a branch, implements, tests, opens a PR
4. Run code review gate — review comments trigger automatic fixes (up to 3 cycles)
5. Record results, move to the next issue
6. Stop when all issues are complete or max expeditions reached
7. Enter D-Mail waiting mode — monitor inbox/ via fsnotify for incoming D-Mails
8. On D-Mail arrival, re-run the expedition loop; on timeout (default 30m), exit

## Why "Paintress"?

The system design is inspired by the world structure of [Clair Obscur: Expedition 33](https://www.expedition33.com/), an RPG game.

In the game world, a being called the Paintress paints a number on a monolith each year, erasing everyone of that age. Every year, the people send an Expedition to destroy her — but every expedition fails. Only their flags and journals remain as guideposts for the next.

This structure maps directly to AI agent loop design:

| Game Concept | Paintress | Design Meaning |
|---|---|---|
| **Paintress** | This binary | External force that drives the loop |
| **Monolith** | Linear backlog | The remaining issue count is inscribed |
| **Expedition** | One Claude Code execution | Departs with fresh context each time |
| **Expedition Flag** | `.expedition/.run/flag.md` | Per-worker checkpoint, consolidated at exit |
| **Journal** | `.expedition/journal/` | Record of past decisions and lessons |
| **Canvas** | LLM context window | Beautiful but temporary — destroyed each run |
| **Lumina** | Auto-extracted patterns | Patterns learned from past failures/successes |
| **Gradient Gauge** | Consecutive success tracker | Momentum unlocks harder challenges |
| **Reserve Party** | Model fallback | When Opus falls, Sonnet takes over |

### Three Design Principles

1. **Always destroy the Canvas** — LLM context is reset every run. A fresh start beats a polluted context.
2. **Plant the Flag well** — Loop quality depends on what you pass to the next Expedition. Checkpoints and Lumina are the lifeline.
3. **Make the Gommage your ally** — Failure (erasure) isn't the end; it's a chance to accumulate Lumina. Consecutive failures trigger a halt and write a defensive insight to `gommage.md` with actual failure reasons extracted from recent journals.

---

## Game Mechanics

Three game mechanics autonomously control loop quality:

### Gradient Gauge (Momentum)

Consecutive successes fill the gauge, unlocking higher-difficulty issues.

```
[░░░░░] 0/5 → Start with small, safe issues
[██░░░] 2/5 → Normal priority
[████░] 4/5 → High priority OK
[█████] 5/5 → GRADIENT ATTACK: tackle the most complex issue
```

- **Success** → +1 (Charge)
- **Skip** → -1 (Decay)
- **Failure** → Reset to 0 (Discharge)

### Lumina (Learned Passive Skills)

Past journals are scanned in parallel goroutines to extract recurring patterns.
Injected directly into the next Expedition's prompt.

- **Defensive**: Insights from failed expeditions that appear 2+ times → "Avoid — failed N times: ..." (falls back to failure reason if no insight)
- **Offensive**: Insights from successful expeditions that appear 3+ times → "Proven approach (Nx successful): ..." (falls back to mission type if no insight)

### Reserve Party (Model Fallback)

The output streaming goroutine detects rate limits in real-time and switches models automatically.

```bash
# Opus primary, Sonnet reserve
paintress --model opus,sonnet ./repo

# Three-tier fallback
paintress --model opus,sonnet,haiku ./repo
```

- Rate limit detected → immediate switch to reserve
- After 30-min cooldown → attempt recovery to primary
- Timeout also triggers reserve switch (possible rate limit)

## D-Mail Protocol

Paintress communicates with external tools (phonewave, sightjack, amadeus) via the D-Mail protocol — Markdown files with YAML frontmatter exchanged through `inbox/` and `outbox/` directories. Each message carries a `dmail-schema-version` field (currently `"1"`) for protocol compatibility.

- **Inbound**: External tools write specification/implementation-feedback d-mails to `inbox/`. Paintress scans and embeds them in the expedition prompt.
- **Pre-Flight Triage**: Before each expedition, `triagePreFlightDMails` processes action fields: `escalate` (consume + emit event), `resolve` (consume + emit resolved event), `retry` (pass through or escalate if over max retries). Triaged-out D-Mails are archived immediately.
- **Outbound**: After a successful expedition, a report d-mail is written to `archive/` first, then `outbox/` (archive-first for durability).
- **HIGH Severity Gate**: HIGH severity d-mails trigger desktop notification + human approval before the expedition starts. See [docs/approval-contract.md](docs/approval-contract.md).
- **Skills**: Agent skill manifests (`SKILL.md`) in `.expedition/skills/` follow the [Agent Skills](https://agentskills.io) specification, declaring D-Mail capabilities under `metadata`.

**BREAKING**: The `feedback` kind has been split into `design-feedback` and `implementation-feedback`. Paintress consumes `implementation-feedback` (not the old `feedback`). Run `paintress doctor` to detect deprecated kinds and `paintress init --force <repo-path>` to regenerate SKILL.md files.

Full protocol details: **[docs/dmail-protocol.md](docs/dmail-protocol.md)** | Directory structure: **[docs/expedition-directory.md](docs/expedition-directory.md)**

## Architecture

```
Paintress (binary)         <- Outside the repository
    |
    |  Pre-flight:
    |  +-- goroutine: parallel journal scan -> Lumina extraction
    |  +-- PreflightCheckRemote (verify git remote exists)
    |  +-- WorktreePool.Init (when --workers >= 1)
    |
    |  Per Expedition:
    |  +-- triagePreFlightDMails (escalate/resolve/retry)
    |  +-- Gradient Gauge check -> difficulty hint
    |  +-- Reserve Party check -> primary recovery attempt
    |
    v
Monolith (Linear)          <- Fully external
    |
    v
WorktreePool               <- Isolated worktrees for parallel workers (Swarm Mode)
    |
    v
Expedition (Claude Code)   <- One session per issue
    |
    v
Review Gate (exec)         <- Code review tool + Claude Code --continue (up to 3 cycles)
    |
    v
D-Mail Waiting Loop        <- fsnotify inbox/ watch (--wait-timeout, default 30m)
    |                         On D-Mail arrival: re-run expedition loop
    |                         On timeout/signal: clean exit
    v
Continent (Git repo)       <- Persistent world
    +-- src/
    +-- CLAUDE.md
    +-- .expedition/
         +-- config.yaml   <- Project config (paintress init)
         +-- journal/
         |    +-- 001.md, 002.md, ...
         +-- context/      <- User-provided .md files injected into prompts
         +-- skills/       <- Agent skill manifests (SKILL.md)
         +-- inbox/        <- Incoming d-mails (gitignored, transient)
         +-- outbox/       <- Outgoing d-mails (gitignored, transient)
         +-- archive/      <- Processed d-mails (tracked, audit trail)
         +-- insights/     <- Insight Ledger (tracked, lumina.md + gommage.md)
         +-- events/       <- Append-only event store (JSONL, gitignored)
         +-- .run/         <- Ephemeral (gitignored)
              +-- flag.md       <- Checkpoint (consolidated from per-worker flags at exit)
              +-- logs/         <- Expedition logs
              +-- worktrees/    <- Managed by WorktreePool
                   +-- worker-001/
                   |    +-- .expedition/.run/flag.md  <- Per-worker checkpoint
                   +-- worker-002/
                        +-- .expedition/.run/flag.md  <- Per-worker checkpoint
```

### WorktreePool Lifecycle (`--workers >= 1`)

1. **Init** — `git worktree prune`, then for each worker: force-remove leftover → `git worktree add --detach` → run `--setup-cmd` if set
2. **Acquire** — Worker claims a worktree from the pool (blocks if all in use)
3. **Release** — After each expedition: `git checkout --detach <base-branch>` → `git reset --hard <base-branch>` → `git clean -fd -e .expedition` → return to pool. The `-e .expedition` exclusion preserves per-worker flag.md across releases.
4. **Consolidate** — After all workers complete: `reconcileFlags` scans all worktree flag.md files, picks max(LastExpedition), writes it back to Continent's flag.md for human inspection and next startup.
5. **Shutdown** — On exit (30s timeout, independent of parent context): `git worktree remove -f` each → `git worktree prune`

When `--workers 0`, no pool is created and expeditions run directly on the repository. The flag.md path unifies: `flagDir = workDir` (worktree path when Workers>0, Continent when Workers=0). No mutex is needed — each worker has exclusive access to its own flag.md. `reconcileFlags` skips worktree glob scan when workers=0, reading only the Continent flag.md to avoid stale worktree contamination from crashed prior runs.

## Goroutines

| Goroutine | Role | Game Concept |
|-----------|------|-------------|
| Signal handler | SIGINT/SIGTERM → context cancel | — |
| Dev server | Background startup & monitoring | Camp |
| Journal scanner | Parallel file reads → Lumina extraction | Resting at Flag |
| Worker (N) | Expedition loop per worktree (Swarm Mode) | Expedition Party |
| Output streaming | stdout tee + rate limit detection | Reserve Party standby |
| Flag watcher | fsnotify: detect issue selection in real-time | Expedition Flag |
| Inbox watcher | fsnotify: detect d-mails arriving mid-expedition | D-Mail courier |
| Timeout watchdog | context.WithTimeout | Gommage (time's up) |

## Code Review Gate

After a successful Expedition creates a PR, Paintress runs an automated code review via a configurable command (default: `codex review --base main`). The review tool is customizable via `--review-cmd` and can be any linter, code review tool, or custom script. The review runs outside the LLM context window to avoid polluting the Expedition's Canvas.

- **Pass**: Review finds no actionable issues → proceed to next Expedition
- **Fail**: Review comments tagged `[P0]`–`[P4]` are detected → Claude Code resumes the Expedition session (`--continue`) to fix them, reusing full implementation context
- **Retry**: Up to 3 review-fix cycles per Expedition; unresolved insights are recorded in the journal
- **Timeout**: The entire review loop is bounded by the expedition timeout (`--timeout`)
- **Rate limit / error**: Review is skipped gracefully (logged as WARN, does not block the loop)

The review command is customizable via `--review-cmd`. Set to empty string (`--review-cmd ""`) to disable.

## Scope

**What Paintress does:**

- Autonomously pick Linear issues and implement code changes via Claude Code
- Create branches, run tests, open PRs, and iterate through code review cycles
- Manage parallel expeditions in isolated git worktrees (Swarm Mode)
- Send report D-Mails to downstream tools after successful expeditions
- Enter D-Mail waiting mode after expeditions complete, re-running on incoming D-Mails

**What Paintress does NOT do:**

- Edit Linear issues directly (only reads issues for implementation)
- Manage git branches on the main repository (uses worktrees for isolation)
- Handle authentication setup (assumes Linear, GitHub CLI, and Claude Code are pre-configured)
- Verify post-merge design integrity (amadeus handles that)

## Setup

```bash
# Install via Homebrew (WIP — tap may not be published yet)
brew install hironow/tap/paintress

# Or build from source
just install

# Initialize project config (Linear team key, etc.)
paintress init /path/to/your/repo

# Upgrade existing project (regenerate SKILL.md, etc.)
paintress init --force /path/to/your/repo

# Check external command availability, git remote, deprecated kinds, context-budget per-item diagnostics
paintress doctor

# Run — .expedition/ is created automatically
paintress /path/to/your/repo
```

Paintress creates `.expedition/` with config, journal entries, and ephemeral
runtime state under `.run/` automatically. Mission and Lumina content are
embedded directly in the expedition prompt (no separate files on disk).
Git worktrees for Swarm Mode are also fully managed — Paintress creates them
on startup and removes them on shutdown. No manual `git worktree` commands needed.

## Subcommands

| Command | Description |
|---------|-------------|
| `paintress <repo-path>` | Run expedition loop (default, `run` subcommand implied) |
| `paintress init <repo-path>` | Initialize `.expedition/config.yaml` interactively (`--force` to regenerate) |
| `paintress doctor` | Check commands, git-remote, deprecated kind detection, Docker CLAUDE_CONFIG_DIR hint, context-budget (per-item diagnostics) |
| `paintress issues <repo-path>` | Query Linear issues via Claude MCP (`-o json` for JSON, `-s` to filter by state) |
| `paintress config show [repo-path]` | Display project configuration |
| `paintress config set <key> <value> [repo-path]` | Update a configuration value (e.g. `tracker.team`, `tracker.project`) |
| `paintress status [repo-path]` | Show paintress operational status |
| `paintress clean <repo-path>` | Remove state directory (`.expedition/`) |
| `paintress rebuild <repo-path>` | Rebuild projections from event store |
| `paintress archive-prune [repo-path]` | Prune old archived d-mails (`-d 14` for days, `-x` to execute) |
| `paintress version` | Show version, commit, date, and Go version (`-j` for JSON) |
| `paintress update` | Self-update to the latest GitHub release (`-C` to check only) |

## Usage

All commands accept an optional `[repo-path]` argument. When omitted, the current working directory is used.

```bash
# Run from the repo directory (cwd fallback)
cd /path/to/repo
paintress run

# Or specify the path explicitly
paintress run /path/to/repo

# Flags support GNU/POSIX long (--flag) and short (-f) forms
paintress run -m opus,sonnet       # short flag
paintress run --model=opus         # --flag=value form
```

```bash
# Basic (Opus only, English prompts)
paintress run

# Japanese prompts
paintress run -l ja

# With Reserve Party
paintress run -m opus,sonnet

# Swarm Mode: 3 parallel workers with setup command
paintress run -m opus,sonnet -w 3 --setup-cmd "bun install"

# Skip dev server (CLI tools, backend-only repos)
paintress run --no-dev

# Dry run (generate prompts only)
paintress run -n

# Prune archived d-mails (dry-run, then execute)
paintress archive-prune
paintress archive-prune -d 14 -x

# Skip code review gate
paintress run --review-cmd ""

# Notify via ntfy.sh when HIGH severity D-Mail arrives
paintress run --notify-cmd 'curl -d "{message}" ntfy.sh/paintress'

# Skip approval gate (CI/automated runs)
paintress run --auto-approve

# Custom waiting timeout (0 = 24h safety cap, negative = disable waiting)
paintress run --wait-timeout 1h

# Disable D-Mail waiting (exit immediately after expeditions)
paintress run --wait-timeout -1s

# Custom approval script
paintress run --approve-cmd './scripts/approve.sh "{message}"'

# All options
paintress run \
  -m opus,sonnet,haiku \
  -l ja \
  --max-expeditions 20 \
  -t 1200 \
  -w 3 \
  --setup-cmd "bun install" \
  --dev-cmd "pnpm dev" \
  --dev-dir /path/to/frontend \
  --dev-url "http://localhost:3000" \
  --review-cmd "codex review --base main" \
  --wait-timeout 30m
```

## Options

### Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--verbose` | `-v` | `false` | Enable verbose output |
| `--output` | `-o` | `text` | Output format: `text` or `json` |
| `--lang` | `-l` | `en` | Prompt language: `en`, `ja`, or `fr` |
| `--version` | | | Show version and exit |

For full flag reference per subcommand, see [docs/cli/](docs/cli/).

## Configuration

Paintress stores project configuration in `.expedition/config.yaml` (generated by `paintress init`). See [docs/expedition-directory.md](docs/expedition-directory.md) for the full directory structure.

## Tracing (OpenTelemetry)

Paintress instruments key operations (expedition, review loop, worktree pool, dev server) with OpenTelemetry spans and events. Tracing is off by default (noop tracer) and activates when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

```bash
# Start Jaeger (all-in-one trace viewer)
docker compose -f docker/compose.yaml up -d

# Run paintress with tracing enabled
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 paintress run

# View traces at http://localhost:16686
```

## Development

All code lives in `internal/` (Go convention). See [docs/conformance.md](docs/conformance.md) for layer architecture and directory responsibilities. Run `just --list` for available tasks.

## Companion Binaries

Paintress ships three companion binaries for sending notifications and approval requests to chat platforms. They plug into `--notify-cmd` and `--approve-cmd`:

| Binary | Platform | Transport | Env Vars |
|--------|----------|-----------|----------|
| `paintress-tg` | Telegram | Bot API (long polling) | `PAINTRESS_TG_TOKEN`, `PAINTRESS_TG_CHAT_ID` |
| `paintress-discord` | Discord | Bot Gateway (WebSocket) | `PAINTRESS_DISCORD_TOKEN`, `PAINTRESS_DISCORD_CHANNEL_ID` |
| `paintress-slack` | Slack | Socket Mode (WebSocket) | `PAINTRESS_SLACK_TOKEN`, `PAINTRESS_SLACK_CHANNEL_ID`, `PAINTRESS_SLACK_APP_TOKEN` |

Each binary provides three subcommands: `notify`, `approve`, and `doctor`.

```bash
# Example: Slack notifications + Telegram approval
paintress \
  --notify-cmd 'paintress-slack notify "{message}"' \
  --approve-cmd 'paintress-tg approve "{message}"' \
  /path/to/repo

# Check companion setup
paintress-tg doctor
paintress-discord doctor
paintress-slack doctor
```

All companions follow the [approval contract](docs/approval-contract.md): exit 0 = approved, exit non-zero = denied.

Build from source: `just install-all` (installs all 4 binaries to `/usr/local/bin`). Homebrew (`brew install hironow/tap/paintress`) is WIP.

## What / Why / How

See [docs/conformance.md](docs/conformance.md) for the full conformance table (single source).

## Documentation

- [docs/](docs/README.md) — Full documentation index
- [docs/conformance.md](docs/conformance.md) — What/Why/How conformance table
- [docs/expedition-directory.md](docs/expedition-directory.md) — `.expedition/` directory structure
- [docs/policies.md](docs/policies.md) — Event → Policy mapping
- [docs/otel-backends.md](docs/otel-backends.md) — OTel backend configuration
- [docs/approval-contract.md](docs/approval-contract.md) — Three-way approval contract
- [docs/testing.md](docs/testing.md) — Test strategy and conventions
- [docs/adr/](docs/adr/README.md) — Architecture Decision Records
- [docs/shared-adr/](docs/shared-adr/README.md) — Cross-tool shared ADRs

## Prerequisites

- Go 1.26+
- [just](https://just.systems/) task runner
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code)
- A code review CLI (for code review gate, customizable via `--review-cmd`, e.g. tools that output `[P0]`–`[P4]` priorities)
- [GitHub CLI](https://cli.github.com/) for Pull Request operations
- Linear: accessible for Issue operations (e.g. Linear MCP)
- [Docker](https://www.docker.com/) for tracing (Jaeger) and container tests
- Browser automation (for verify missions): e.g. Playwright, Chrome DevTools

## License

Apache License 2.0
See [LICENSE](./LICENSE) for details.
