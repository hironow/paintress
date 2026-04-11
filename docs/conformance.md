# What / Why / How Conformance

This is the single source of truth for paintress's purpose, design rationale, and implementation approach.
Referenced from [README.md](../README.md) and [docs/README.md](README.md).

| Aspect | Description |
|--------|-------------|
| **What** | Autonomous loop that sends AI agents to implement Linear issues until the backlog reaches zero |
| **Why** | Automate the full issue lifecycle: implement, test, PR, review, fix — with no human intervention |
| **How** | Fetch issue via Claude MCP → Claude Code implements in git branch → review gate → PR → next issue → enter D-Mail waiting loop (fsnotify inbox/ watch, re-run expeditions on arrival, default 30m timeout, 0=24h safety cap) |
| **Input** | Linear issues via Claude MCP tools, repository source code, inbox D-Mails (specification, implementation-feedback) |
| **Output** | Git branches, pull requests, D-Mail reports to downstream tools |
| **Telemetry** | OTel spans: `paintress.run`, `expedition`, `provider.invoke` (with `provider.model`, `provider.timeout_sec`, `gen_ai.*`), `context_budget.*` (`context_budget.tools`, `context_budget.skills`, `context_budget.plugins`, `context_budget.mcp_servers`, `context_budget.hook_bytes`, `context_budget.estimated_tokens`) |
| **External Systems** | Linear (via Claude MCP), Claude Code subprocess, Git/GitHub, OTel exporter (Jaeger/Weave) |

## Layer Architecture

```
cmd              --> usecase, session, harness, usecase/port, platform, domain  (composition root)
usecase          --> usecase/port, harness, domain                              (output port only)
usecase/port     --> domain (+ stdlib)                                          (interface contracts)
session          --> eventsource, harness, usecase/port, platform, domain       (adapter impl)
harness          --> domain                                                     (policy/verifier/filter facade)
  harness/policy   --> domain                                                   (deterministic decisions)
  harness/verifier --> domain, harness/policy                                   (validation)
  harness/filter   --> domain, harness/verifier, harness/policy                 (prompt rendering & LLM action space)
eventsource      --> domain                                                     (event persistence adapter)
platform         --> domain (+ stdlib)                                          (cross-cutting infra)
domain           --> (nothing internal, stdlib only)                             (pure types/logic)
```

`harness` is the decision/validation/specification layer between the LLM and the environment.
It is organized as three sub-packages behind a facade (`harness.go` re-exports all):

- **`harness/policy`** — Deterministic decision logic (GradientGauge, ReserveParty, RetryTracker, wave projection, stagnation detection, strategy rotation, reflection accumulator, expedition target selection)
- **`harness/verifier`** — Validation of LLM outputs and external signals (review comment detection, rate limit detection, D-Mail schema validation, provider error classification)
- **`harness/filter`** — Prompt construction and LLM action space (PromptRegistry, expedition/review/D-Mail/Lumina/reflection prompt rendering, strategy text, prompt optimizer)

External callers (cmd, usecase, session) must use the `harness` facade — direct imports of sub-packages are prohibited by semgrep. Sub-packages must not import the facade (circular dependency prevention).

`eventsource` is the event persistence adapter based on the [AWS Event Sourcing pattern](https://docs.aws.amazon.com/prescriptive-guidance/latest/cloud-design-patterns/event-sourcing.html).
Its responsibility is limited to append, load, and replay of domain events.
Event store implementation MUST NOT exist outside `internal/eventsource`.
`session` uses `eventsource` as a client but does not implement event persistence itself.

Key constraints enforced by semgrep (ERROR severity):

- `usecase --> session` PROHIBITED (must use output port interfaces)
- `cmd --> eventsource` PROHIBITED (ADR S0008)
- `domain` has no I/O, no `context.Context`
- `domain --> harness` PROHIBITED (domain is pure types/logic)
- `eventsource --> harness` PROHIBITED
- `harness/policy` may only import `domain`
- `harness/verifier` may only import `domain` + `harness/policy`
- `harness/filter` may only import `domain` + `harness/verifier` + `harness/policy`
- External callers must use `harness` facade, not sub-packages directly

Ref: `.semgrep/layers.yaml`, `.semgrep/layers-harness.yaml`, `refs/opsx/semgrep-layer-contract.md`, ADR S0007

## Domain Primitives & Parse-Don't-Validate

Domain command types use the Parse-Don't-Validate pattern:

- Domain primitives (`RepoPath`, `Days`, `Team`, `Project`) validate in `New*()` constructors — invalid values are rejected at parse time
- Command types use unexported fields with `New*Command()` constructors that accept only pre-validated primitives
- Commands are always-valid by construction — no `Validate() []error` methods exist
- Usecase layer receives always-valid commands with no validation boilerplate
- Semgrep rule `domain-no-validate-method` prevents reintroduction of `Validate() []error`

Ref: `.semgrep/layers.yaml`, ADR S0029

## Tracking Mode (Wave vs Linear)

### Claude Subprocess Isolation

Claude subprocess uses layered isolation to prevent parent session context (266+ skills, 66+ plugins) from inflating token usage:

- `--setting-sources ""` skips all user/project settings (hooks, plugins, auto-memory) while preserving OAuth authentication
- `--settings <stateDir>/.claude/settings.json` loads tool-specific settings (empty `enabledPlugins`)
- `--disable-slash-commands` prevents user skills from inflating context
- `--strict-mcp-config --mcp-config <stateDir>/.mcp.json` enforces MCP server allowlist
- `mcp-config generate` creates both `.mcp.json` (wave: empty, linear: Linear MCP) and `.claude/settings.json`
- User can edit `.mcp.json` to add custom MCP servers, `.claude/settings.json` for env vars or permissions

### Claude Log Persistence

- `WriteClaudeLog` saves raw NDJSON to `.run/claude-logs/{timestamp}.jsonl` after each invocation
- Enables post-hoc debugging and audit of Claude subprocess interactions
- Managed by archive-prune lifecycle

- **Wave mode** (default, `--linear` not set): `ProjectWaveState()` builds wave progress from D-Mail archive. `ExpeditionTargetsFromWaves()` extracts pending steps as expedition targets. Claim keys use step IDs. Prompt templates inject step title and acceptance criteria. Report D-Mails include `WaveReference{ID, Step}` for archive projection.
- **Linear mode** (`--linear`): Existing behavior — `FetchIssuesViaMCP` via Linear MCP, issue-based claims, Linear team/project in prompts.
- `TargetProvider` port interface abstracts the source; `WaveTargetProvider` (wave) and `FetchIssuesViaMCP` (linear) are the two implementations.

Ref: ADR S0035, `internal/domain/wave_projection.go`, `internal/domain/expedition_target.go`, `internal/usecase/wave_targets.go`

## Cross-Tool Conformance

All 4 tools (phonewave, sightjack, paintress, amadeus) maintain a What/Why/How conformance table in `docs/conformance.md` with the same structure. This prevents expression drift across README files.

## Harness Inventory (Track A)

| Sub-package | Key functions | Role |
|-------------|---------------|------|
| `harness/policy` | `Preflight`, `Gradient`, `Reserve`, `RetryTracker`, `ExpeditionTarget`, `Review`, `EvaluateExhaustion`, `RunGuard`, `ReflectionAccumulator` | Deterministic decisions |
| `harness/verifier` | `ProviderError`, `Review`, `DMail` | Validation rules |
| `harness/filter` | `Expedition`, `Lumina`, `Reflection`, `Optimizer`, `DMail` | LLM action spaces |

Ref: ADR S0038, S0039

## Improvement Controller (Track D3/F)

The improvement controller resides in amadeus (ADR S0041). paintress receives corrective D-Mails as a consumer and forwards outcomes via report D-Mails.
