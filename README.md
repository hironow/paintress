# Paintress

**An MCP server + data plane for the Expedition workflow: it serves the expedition journal/gradient read models from the session's continent dir and persists expedition-completed / gradient events.**

Following the jun15 MCP pivot, LLM ownership moved to a human-initiated [Claude Code](https://docs.anthropic.com/en/docs/claude-code) session. Paintress the Go CLI is now a pure data plane: it serves completed-issue ids and the next expedition number over MCP, persists gradient + expedition-completed events to the event store, and provides the supporting data-plane commands (init, doctor, status, sessions, archive-prune, ...). The headless autonomous loop — the `claude --print` subprocess, swarm worktree pool, review gate, gommage recovery, and D-Mail composition — has been retired. The Expedition workflow now fires from the claude-code session via the `/expedition-next` skill and the paintress MCP tools (see `plugins/paintress/skills/expedition-next/SKILL.md`).

```bash
paintress mcp
```

`paintress mcp` starts the MCP server (over stdio, embedded via `--mcp-config`). Its tools expose:

1. `paintress.ping` — health check
2. `paintress.next_issue` — reads `pr-index.jsonl` + `journal/` to surface completed issue ids + the next expedition number
3. `paintress.update_gradient` — persists a gradient-changed event to the event store
4. `paintress.append_journal` — persists an expedition-completed event (journal + pr-index write)

The claude-code session reads these read models, runs the expedition itself (implement / verify / fix, branch + PR), and writes report D-Mails to `outbox/` via the skill workflow — paintress no longer drives the LLM or composes D-Mails. Inference stays on the session's subscription quota rather than crossing into the Agent SDK credit pool that gates `claude --print` from 2026-06-15.

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
3. **Make the Gommage your ally** — Failure (erasure) isn't the end; it's a chance to accumulate Lumina. Consecutive failures trigger class-aware recovery: transient failures (timeout, rate limit, parse error) retry with cooldown, while persistent failures (blocker, systematic) halt and escalate. Recovery resets counters, injects Lumina hints, and starts a fresh attempt for the same issue type.

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

### Reserve Party (Model Cascade Fallback)

The output streaming goroutine detects rate limits in real-time and cascades through available models automatically. Each model has an independent 30-minute cooldown, so a three-tier configuration can fall back from Opus to Sonnet to Haiku without waiting.

```bash
# Opus primary, Sonnet reserve
paintress --model opus,sonnet ./repo

# Three-tier cascade fallback
paintress --model opus,sonnet,haiku ./repo
```

- Rate limit detected → put current model in per-model cooldown → switch to next available model
- After 30-min cooldown expires → attempt recovery to primary
- Timeout also triggers cascade switch (possible rate limit)

## Expedition Intelligence

Additional systems that improve expedition quality across runs:

### Capability Detection

`ClassifyCapabilityViolation` scans journal text for signals indicating the expedition hit an environment boundary (network access, filesystem permissions, missing tools, Docker unavailability, auth failures, resource limits). Detected violations are recorded and injected into the Capability Boundary section of subsequent expedition prompts to prevent repeated failures.

### Reflection Accumulator

`ReflectionAccumulator` collects review comments across review-fix cycles within a single expedition. It tracks priority tag counts per cycle and detects stagnation (tag counts not decreasing across cycles). `FormatForPrompt` renders the accumulated history for injection into fix prompts.

### Strategy Rotation

`StrategyForCycle` rotates through three fix strategies across review-fix cycles: **Direct** (cycle 1) applies review comments directly, **Decompose** (cycle 2) breaks comments into sub-tasks, **Rewrite** (cycle 3) rewrites the affected section from scratch. The rotation repeats for longer review chains.

### Issue Claim Registry

`IssueClaimRegistry` prevents multiple parallel workers (Swarm Mode) from working on the same Linear issue simultaneously. Thread-safe via mutex; `TryClaim` returns the holding expedition number on conflict.

### Duration Percentiles

`ExpeditionDurations` pairs start/complete events to compute per-expedition durations. `DurationPercentiles` calculates p50, p90, and p99 from the duration list. Telemetry breakdown attributes track time spent in each expedition phase.

### Windowed Success Rate

`WindowedSuccessRate` computes success rate over the most recent N completed expeditions. `DetectSuccessRateTrend` compares the recent window against the preceding window to detect improvement, decline, or stability (threshold: 10% change).

### Worktree Health Check

`AcquireContext` runs a `git status` health check on acquired worktrees before returning them to workers. If the worktree is corrupted or inaccessible, it is automatically force-recycled and a fresh worktree is created. Acquired worktrees are also cleaned up on `Shutdown`.

### Context File Size Guard

Per-file and total byte limits prevent oversized context files from bloating the expedition prompt. Files exceeding the per-file limit are excluded with a warning; the total budget caps aggregate context size.

### Review Comment Extraction

`ExtractReviewComments` parses review tool output into structured `ReviewComment` values with priority sorting (`[P0]` highest). Falls back to raw text when structured parsing fails.

### Escalation Cooldown

Escalation events fire once per failure streak rather than on every consecutive failure. Retry backoff is capped via `NewRetryTrackerWithMax` with an `Exhausted` check.

### Label-Based Issue Exclusion

`ExcludeIssuesByLabel` filters Linear issues by label (case-insensitive match), allowing teams to exclude issues tagged with specific labels from the expedition loop.

## D-Mail Protocol

Paintress communicates with external tools (phonewave, sightjack, amadeus) via the D-Mail protocol — Markdown files with YAML frontmatter exchanged through `inbox/` and `outbox/` directories. Each message carries a `dmail-schema-version` field (currently `"1"`) for protocol compatibility.

- **Inbound**: External tools write specification/implementation-feedback d-mails to `inbox/`. Paintress scans and embeds them in the expedition prompt.
- **Pre-Flight Triage**: Before each expedition, `triagePreFlightDMails` processes action fields: `escalate` (consume + emit event), `resolve` (consume + emit resolved event), `retry` (pass through or escalate if over max retries). Triaged-out D-Mails are archived immediately.
- **Outbound**: After a successful expedition, a report d-mail is written to `archive/` first, then `outbox/` (archive-first for durability).
- **HIGH Severity Gate**: HIGH severity d-mails trigger desktop notification + human approval before the expedition starts. See [docs/approval-contract.md](docs/approval-contract.md).
- **Skills**: Agent skill manifests (`SKILL.md`) in `.expedition/skills/` follow the [Agent Skills](https://agentskills.io) specification, declaring D-Mail capabilities under `metadata`.

**BREAKING**: The `feedback` kind has been split into `design-feedback` and `implementation-feedback`. Paintress consumes `implementation-feedback` (not the old `feedback`). Run `paintress doctor` to detect deprecated kinds and `paintress init --force [path]` to regenerate SKILL.md files.

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
    |  +-- IssueClaimRegistry.TryClaim (Swarm Mode dedup)
    |  +-- triagePreFlightDMails (escalate/resolve/retry)
    |  +-- Gradient Gauge check -> difficulty hint
    |  +-- Reserve Party check -> primary recovery attempt
    |  +-- StrategyForCycle -> fix strategy selection
    |  +-- ReflectionAccumulator -> stagnation detection
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
D-Mail Waiting Loop        <- fsnotify inbox/ watch (--idle-timeout, default 30m)
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
2. **Acquire** — Worker claims a worktree from the pool (blocks if all in use). `AcquireContext` runs a `git status` health check; corrupted worktrees are force-recycled and re-created automatically.
3. **Release** — After each expedition: `git checkout --detach <base-branch>` → `git reset --hard <base-branch>` → `git clean -fd -e .expedition` → return to pool. The `-e .expedition` exclusion preserves per-worker flag.md across releases. Checkout/reset failures trigger automatic worktree recycling.
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

## Scope

**What Paintress does (post jun15 MCP pivot):**

- Serve the expedition journal/gradient read models over MCP (`next_issue`) to a claude-code session
- Persist gradient-changed + expedition-completed events to the event store (`update_gradient` / `append_journal`)
- Provide the supporting data-plane commands (init, doctor, status, sessions, archive-prune, rebuild, dead-letters)
- Generate the claude-code MCP wiring (`mcp-config generate`)

The expedition workflow itself (pick an issue, implement, test, open a PR, send report D-Mails) now runs inside the claude-code session via the `/expedition-next` skill — paintress no longer drives the LLM, runs a swarm worktree pool, or composes D-Mails.

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

# Generate Claude subprocess isolation settings
paintress mcp-config generate /path/to/your/repo

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

The headless `run` and `issues` subcommands were retired by the jun15 MCP pivot. The expedition workflow now runs from a claude-code session via the `/expedition-next` skill + the `paintress mcp` data-plane server.

| Command | Description |
|---------|-------------|
| `mcp` | Run the MCP server over stdio (expedition journal/gradient data plane) |
| `init` | Initialize `.expedition/config.yaml` |
| `doctor` | Check environment health |
| `sessions` / `sessions enter` / `sessions list` | Inspect and enter recorded coding sessions |
| `config show` / `config set` | View or update configuration |
| `status` | Show operational status |
| `clean` | Remove state directory |
| `rebuild` | Rebuild projections from event store |
| `archive-prune` | Prune old archived D-Mail files |
| `dead-letters` | Inspect / purge dead-letter D-Mails |
| `version` | Print version info |
| `mcp-config generate` | Generate `.mcp.json` and `.claude/settings.json` for the claude-code session |
| `update` | Self-update to the latest release |

All commands accept an optional `[path]` argument (defaults to cwd). For flags, examples, and full reference per subcommand, see [docs/cli/](docs/cli/).

## Quick Start

```bash
paintress init                          # set up .expedition/
paintress mcp-config generate           # claude-code MCP wiring
paintress mcp                           # start the data-plane MCP server (embed via --mcp-config)
```

Then run the Expedition workflow from a claude-code session via the `/expedition-next` skill.

## Configuration

Paintress stores project configuration in `.expedition/config.yaml` (generated by `paintress init`). See [docs/expedition-directory.md](docs/expedition-directory.md) for the full directory structure.

## Tracing (OpenTelemetry)

Paintress instruments key operations (expedition, review loop, worktree pool, dev server) with OpenTelemetry spans and events. Tracing is off by default (noop tracer) and activates when `OTEL_EXPORTER_OTLP_ENDPOINT` is set.

```bash
# Start Jaeger (all-in-one trace viewer)
docker compose -f docker/compose.yaml up -d

# Run the paintress MCP server with tracing enabled
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 paintress mcp

# View traces at http://localhost:16686
```

## Development

All code lives in `internal/` (Go convention). The `internal/harness/` layer provides the decision/validation/prompt-rendering boundary between the LLM and the environment, organized as `policy/` (deterministic decisions), `verifier/` (output validation), and `filter/` (prompt construction) behind a single facade. See [docs/conformance.md](docs/conformance.md) for the full layer architecture, dependency rules, and directory responsibilities. Run `just --list` for available tasks.

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
