# Paintress

**An autonomous loop that sends AI agents to drain your Linear issues to zero.**

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
6. Stop when all issues are complete

## Why "Paintress"?

The system design is inspired by the world structure of [Clair Obscur: Expedition 33](https://www.expedition33.com/), an RPG game.

In the game world, a being called the Paintress paints a number on a monolith each year, erasing everyone of that age. Every year, the people send an Expedition to destroy her — but every expedition fails. Only their flags and journals remain as guideposts for the next.

This structure maps directly to AI agent loop design:

| Game Concept | Paintress | Design Meaning |
|---|---|---|
| **Paintress** | This binary | External force that drives the loop |
| **Monolith** | Linear backlog | The remaining issue count is inscribed |
| **Expedition** | One Claude Code execution | Departs with fresh context each time |
| **Expedition Flag** | `.expedition/.run/flag.md` | Checkpoint passed to the next run |
| **Journal** | `.expedition/journal/` | Record of past decisions and lessons |
| **Canvas** | LLM context window | Beautiful but temporary — destroyed each run |
| **Lumina** | Auto-extracted patterns | Patterns learned from past failures/successes |
| **Gradient Gauge** | Consecutive success tracker | Momentum unlocks harder challenges |
| **Reserve Party** | Model fallback | When Opus falls, Sonnet takes over |

### Three Design Principles

1. **Always destroy the Canvas** — LLM context is reset every run. A fresh start beats a polluted context.
2. **Plant the Flag well** — Loop quality depends on what you pass to the next Expedition. Checkpoints and Lumina are the lifeline.
3. **Make the Gommage your ally** — Failure (erasure) isn't the end; it's a chance to accumulate Lumina. Consecutive failures trigger a halt.

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

Paintress communicates with external tools (phonewave, sightjack, courier) via the D-Mail protocol — markdown files exchanged through `inbox/` and `outbox/` directories.

**Inbound** (inbox/ → prompt): External tools write specification or feedback d-mails to `.expedition/inbox/`. Paintress scans them at expedition start (`ScanInbox`) and embeds them in the prompt. A real-time `watchInbox` goroutine (fsnotify) also detects d-mails arriving mid-expedition (logged but not processed until the next expedition).

**Outbound** (report → outbox/): After a successful expedition, Paintress generates a report d-mail and writes it to `archive/` first, then `outbox/` (archive-first for durability). The courier tool picks up outbox/ files for delivery.

**Lifecycle**: inbox/ → prompt injection → expedition → archive/ (processed). Mid-expedition arrivals stay in inbox/ for the next expedition. Only d-mails that were embedded in the prompt are archived.

**Skills**: Agent skill manifests (`SKILL.md`) in `.expedition/skills/` declare D-Mail capabilities (`dmail-readable`: consumes specification/feedback, `dmail-sendable`: produces reports).

## Architecture

```
Paintress (binary)         <- Outside the repository
    |
    |  Pre-flight:
    |  +-- goroutine: parallel journal scan -> Lumina extraction
    |  +-- WorktreePool.Init (when --workers >= 1)
    |
    |  Per Expedition:
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
Review Gate (exec)         <- Codex review + Claude Code --continue (up to 3 cycles)
    |
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
         +-- .run/         <- Ephemeral (gitignored)
              +-- flag.md       <- Checkpoint (auto-generated)
              +-- logs/         <- Expedition logs
              +-- worktrees/    <- Managed by WorktreePool
```

### WorktreePool Lifecycle (`--workers >= 1`)

1. **Init** — `git worktree prune`, then for each worker: force-remove leftover → `git worktree add --detach` → run `--setup-cmd` if set
2. **Acquire** — Worker claims a worktree from the pool (blocks if all in use)
3. **Release** — After each expedition: `git checkout --detach <base-branch>` → `git reset --hard <base-branch>` → `git clean -fd` → return to pool
4. **Shutdown** — On exit (30s timeout, independent of parent context): `git worktree remove -f` each → `git worktree prune`

When `--workers 0`, no pool is created and expeditions run directly on the repository.

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

After a successful Expedition creates a PR, Paintress runs an automated code review using [Codex CLI](https://github.com/openai/codex) (default: `codex review --base main`). The review itself runs outside the LLM context window to avoid polluting the Expedition's Canvas.

- **Pass**: Review finds no actionable issues → proceed to next Expedition
- **Fail**: Review comments tagged `[P0]`–`[P4]` are detected → Claude Code resumes the Expedition session (`--continue`) to fix them, reusing full implementation context
- **Retry**: Up to 3 review-fix cycles per Expedition; unresolved insights are recorded in the journal
- **Timeout**: The entire review loop is bounded by the expedition timeout (`--timeout`)
- **Rate limit / error**: Review is skipped gracefully (logged as WARN, does not block the loop)

The review command is customizable via `--review-cmd`. Set to empty string (`--review-cmd ""`) to disable.

## Setup

```bash
# Build and install
just install

# Initialize project config (Linear team key, etc.)
paintress init /path/to/your/repo

# Check external command availability
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
| `paintress <repo-path>` | Run expedition loop (default) |
| `paintress init <repo-path>` | Initialize `.expedition/config.yaml` interactively |
| `paintress doctor` | Check required external commands (git, claude, gh, docker) |
| `paintress issues <repo-path>` | List Linear issues (`--output json` for JSON, `--state` to filter) |
| `paintress --version` | Show version and exit |

## Usage

Flags and repo path can be placed in any order:

```bash
paintress --lang ja .          # flags before path
paintress . --lang ja          # flags after path
paintress --model=opus .       # --flag=value form
paintress -- ./my-repo         # -- terminates flags
```

```bash
# Basic (Opus only, English prompts)
paintress /path/to/repo

# Japanese prompts
paintress --lang ja /path/to/repo

# French prompts
paintress --lang fr /path/to/repo

# With Reserve Party
paintress --model opus,sonnet /path/to/repo

# Direct execution (no worktrees, run on repo directly)
paintress --workers 0 /path/to/repo

# Swarm Mode: 3 parallel workers with setup command
paintress \
  --workers 3 \
  --setup-cmd "bun install" \
  --model opus,sonnet \
  /path/to/repo

# All options
paintress \
  --model opus,sonnet,haiku \
  --lang ja \
  --max-expeditions 20 \
  --timeout 1200 \
  --workers 3 \
  --setup-cmd "bun install" \
  --dev-cmd "pnpm dev" \
  --dev-dir /path/to/frontend \
  --dev-url "http://localhost:3000" \
  --review-cmd "codex review --base main" \
  /path/to/repo

# Skip dev server (CLI tools, backend-only repos)
paintress --no-dev /path/to/repo

# Skip code review gate
paintress --review-cmd "" /path/to/repo

# Dry run (generate prompts only)
paintress --dry-run /path/to/repo

# Custom Claude CLI path (e.g. when using an alias with env vars)
# Shell aliases like `alias cc-p="CLAUDE_CONFIG_DIR=~/.claude claude"`
# don't work with exec.Command — specify the binary path and env separately:
CLAUDE_CONFIG_DIR=~/.claude paintress \
  --claude-cmd ~/.local/bin/claude \
  /path/to/repo

# Dev server in a different directory than the repo
paintress \
  --dev-cmd "bun run dev" \
  --dev-dir /path/to/frontend/app \
  --dev-url "http://localhost:5174" \
  /path/to/repo
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `opus` | Model(s), comma-separated for Reserve Party |
| `--lang` | `en` | Prompt language: `en`, `ja`, or `fr` |
| `--max-expeditions` | `50` | Maximum number of expeditions |
| `--timeout` | `1980` | Timeout per expedition and review loop in seconds (33 min) |
| `--base-branch` | `main` | Base git branch |
| `--claude-cmd` | `claude` | Claude Code CLI command name |
| `--dev-cmd` | `npm run dev` | Dev server command |
| `--dev-dir` | repo path | Dev server working directory |
| `--dev-url` | `http://localhost:3000` | Dev server URL |
| `--review-cmd` | `codex review --base main` | Code review command after PR creation |
| `--workers` | `1` | Number of parallel expedition workers (`0` = direct execution without worktrees, `1` = single worktree, `2+` = Swarm Mode) |
| `--setup-cmd` | `""` | Command to run after worktree creation (e.g. `bun install`) |
| `--no-dev` | `false` | Skip dev server startup entirely |
| `--dry-run` | `false` | Generate prompts without executing |
| `--output` | `text` | Output format: `text` or `json` (streaming goes to stderr in JSON mode) |
| `--version` | — | Show version and exit |

## Tracing (OpenTelemetry)

Paintress instruments key operations (expedition, review loop, worktree pool, dev server) with OpenTelemetry spans and events. Tracing is off by default (noop tracer) and activates when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

```bash
# Start Jaeger (all-in-one trace viewer)
just jaeger

# Run paintress with tracing enabled
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 paintress ./your-repo

# View traces at http://localhost:16686

# Stop Jaeger
just jaeger-down
```

## Development

```bash
# Task runner (just)
just build          # Build binary
just install        # Build and install to /usr/local/bin
just test           # Run all tests
just test-v         # Verbose test output
just test-race      # Tests with race detector
just cover          # Coverage report
just cover-html     # Open coverage in browser
just fmt            # Format code (gofmt)
just vet            # Run go vet
just lint           # fmt check + go vet + markdown lint
just lint-md        # Lint markdown files only
just check          # fmt + vet + test (pre-commit check)
just clean          # Clean build artifacts
just prek-install   # Install prek hooks (pre-commit + pre-push)
just prek-run       # Run all prek hooks on all files
just jaeger         # Start Jaeger trace viewer (docker)
just jaeger-down    # Stop Jaeger
```

## File Structure

```
+-- cmd/paintress/
|   +-- main.go              CLI entry point + flag parsing
|   +-- main_test.go         CLI arg parsing tests
+-- main.go                  Config struct + ValidateContinent (library)
+-- paintress.go             Gommage loop
+-- expedition.go            Single Expedition + prompt generation
+-- gradient.go              Gradient Gauge
+-- lumina.go                Lumina scanning (goroutines)
+-- reserve.go               Reserve Party (goroutine)
+-- devserver.go             Dev server (goroutine)
+-- flag.go                  Flag read/write
+-- flag_watcher.go          Real-time issue selection watcher (fsnotify)
+-- inbox_watcher.go         Real-time inbox d-mail watcher (fsnotify)
+-- journal.go               Journal read/write
+-- report.go                Report parser (including failure_type)
+-- context.go               Context injection (.expedition/context/)
+-- dmail.go                 D-Mail protocol (scan, send, archive, parse)
+-- issues.go                Linear issue fetcher (API + formatting)
+-- worktree.go              WorktreePool for Swarm Mode
+-- review.go                Code review gate (exec + parse)
+-- mission.go               Mission text (prompt-embedded)
+-- project_config.go        Project config (.expedition/config.yaml)
+-- init.go                  Interactive init flow
+-- doctor.go                External command checker
+-- telemetry.go             OpenTelemetry tracer setup
+-- lang.go                  i18n message map (en/ja/fr)
+-- logger.go                Colored logging
+-- *_test.go                Tests
+-- justfile                 Task runner
+-- docker/
|   +-- compose.yaml         Jaeger all-in-one for trace viewing
+-- templates/
    +-- expedition_*.md.tmpl Expedition prompt (en/ja/fr)
    +-- mission_*.md.tmpl    Mission rules (en/ja/fr)
    +-- skills/              Agent skill manifests (copied to .expedition/skills/)
        +-- dmail-sendable/SKILL.md
        +-- dmail-readable/SKILL.md
```

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code)
- A code review CLI (for code review gate, customizable via `--review-cmd`, e.g. tools that output `[P0]`–`[P4]` priorities)
- [GitHub CLI](https://cli.github.com/) for Pull Request operations
- Linear: accessible for Issue operations (e.g. Linear MCP)
- [Docker](https://www.docker.com/) for tracing (Jaeger) and container tests
- Browser automation (for verify missions): e.g. Playwright, Chrome DevTools

## License

Apache License 2.0
See [LICENSE](./LICENSE) for details.
