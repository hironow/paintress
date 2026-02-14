# Paintress

**An autonomous loop that sends AI agents to drain your Linear issues to zero.**

Paintress uses [Claude Code](https://docs.anthropic.com/en/docs/claude-code) to automatically process Linear issues one by one — implementing code, running tests, creating PRs, running code reviews, verifying UI, and fixing bugs — with no human intervention, until every issue is done.

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
| **Expedition Flag** | `.expedition/flag.md` | Checkpoint passed to the next run |
| **Journal** | `.expedition/journal/` | Record of past decisions and lessons |
| **Canvas** | LLM context window | Beautiful but temporary — destroyed each run |
| **Lumina** | Auto-extracted patterns | Wisdom learned from past failures/successes |
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
Saved to `.expedition/lumina.md` and injected into the next Expedition's prompt.

- **Defensive**: Insights from failed expeditions that appear 2+ times → "Watch out for this pattern" (falls back to failure reason if no insight)
- **Offensive**: Insights from successful expeditions that appear 3+ times → "This approach is reliable" (falls back to mission type if no insight)

### Reserve Party (Model Fallback)

The output streaming goroutine detects rate limits in real-time and switches models automatically.

```bash
# Opus primary, Sonnet reserve
paintress --model opus,sonnet ./repo

# Three-tier fallback
paintress --model opus,sonnet,haiku ./repo
```

- Rate limit detected → immediate switch to reserve
- After 33-min cooldown → attempt recovery to primary
- Timeout also triggers reserve switch (possible rate limit)

## Architecture

```
Paintress (binary)         <- Outside the repository
    |
    |  Before each Expedition:
    |  +-- goroutine: parallel journal scan -> Lumina extraction
    |  +-- Gradient Gauge check -> difficulty hint
    |  +-- Reserve Party check -> primary recovery attempt
    |
    v
Monolith (Linear)          <- Fully external
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
         +-- flag.md       <- Checkpoint (auto-generated)
         +-- mission.md    <- Rules of engagement (auto-generated, --lang aware)
         +-- lumina.md     <- Learned skills (auto-generated)
         +-- journal/
              +-- 001.md, 002.md, ...
```

## Goroutines

| Goroutine | Role | Game Concept |
|-----------|------|-------------|
| Signal handler | SIGINT/SIGTERM → context cancel | — |
| Dev server | Background startup & monitoring | Camp |
| Journal scanner | Parallel file reads → Lumina extraction | Resting at Flag |
| Output streaming | stdout tee + rate limit detection | Reserve Party standby |
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

# Run — .expedition/ is created automatically
paintress /path/to/your/repo
```

Paintress creates `.expedition/` and all files (`flag.md`, `mission.md`,
`lumina.md`, journal entries) automatically at runtime.

## Usage

**Important:** The repo path (`<repo-path>`) must come **after** all flags.
Go's `flag` package stops parsing at the first non-flag argument, so flags
placed after the repo path will be ignored.

```bash
# OK  — flags first, repo path (or ".") last
paintress --lang ja .

# BAD — flags after repo path are ignored
paintress . --lang ja
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

# All options
paintress \
  --model opus,sonnet,haiku \
  --lang ja \
  --max-expeditions 20 \
  --timeout 1200 \
  --dev-cmd "pnpm dev" \
  --dev-dir /path/to/frontend \
  --dev-url "http://localhost:3000" \
  --review-cmd "codex review --base main" \
  /path/to/repo

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
| `--dry-run` | `false` | Generate prompts without executing |

## Development

```bash
# Task runner (just)
just build          # Build binary
just install        # Build and install to /usr/local/bin
just test           # Run all tests
just test-v         # Verbose test output
just test-race      # Tests with race detector
just cover          # Coverage report
just lint           # Format check + go vet
just check          # fmt + vet + test (pre-commit check)
```

## File Structure

```
+-- main.go              CLI + signal handling
+-- paintress.go         Gommage loop
+-- expedition.go        Single Expedition + prompt generation
+-- gradient.go          Gradient Gauge
+-- lumina.go            Lumina scanning (goroutines)
+-- reserve.go           Reserve Party (goroutine)
+-- devserver.go         Dev server (goroutine)
+-- flag.go              Flag read/write
+-- journal.go           Journal read/write
+-- report.go            Report parser
+-- review.go            Code review gate (exec + parse)
+-- mission.go           Mission writer (embed + template)
+-- lang.go              i18n message map (en/ja/fr)
+-- logger.go            Colored logging
+-- *_test.go            Tests
+-- justfile             Task runner
+-- templates/
    +-- expedition_*.md.tmpl  Expedition prompt (en/ja/fr)
    +-- mission_*.md.tmpl     Mission rules (en/ja/fr)
```

## Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code)
- [Codex CLI](https://github.com/openai/codex) (for code review gate, customizable via `--review-cmd`)
- GitHub: accessible for Pull Request operations (e.g. [GitHub CLI](https://cli.github.com/))
- Linear: accessible for Issue operations (e.g. Linear MCP)
- Browser automation (for verify missions): e.g. Playwright, Chrome DevTools

## License

Apache License 2.0
See [LICENSE](./LICENSE) for details.
