# What / Why / How Conformance

This is the single source of truth for paintress's purpose, design rationale, and implementation approach.
Referenced from [README.md](../README.md) and [docs/README.md](README.md).

| Aspect | Description |
|--------|-------------|
| **What** | MCP server + data plane for the Expedition workflow: serves journal/gradient read models and records expedition events |
| **Why** | Let a human-initiated claude-code session run implementation work while the Go CLI owns durable local projections and event persistence |
| **How** | `paintress mcp` serves MCP tools (`next_issue`, `update_gradient`, `append_journal`); the `/expedition-next` skill in the claude-code session owns issue choice, implementation, verification, PR creation, and D-Mail composition |
| **Input** | `.expedition/` config, journal, event store, PR index, MCP tool arguments |
| **Output** | MCP tool responses, gradient events, expedition-completed events, journal entries, PR index updates |
| **Telemetry** | OTel spans on command roots and MCP tool handlers; `context_budget.*` attributes remain available for recorded claude-code session metadata |
| **External Systems** | Local filesystem, Git/GitHub via the claude-code session, OTel exporter (Jaeger/Weave), claude-code session as MCP client |

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

## MCP Pivot Boundary

Paintress does not own model inference, manage a worktree swarm, run review gates, or compose D-Mails from the Go CLI. LLM execution and repository modification are owned by a human-initiated Claude Code session attached to `paintress mcp`.

- `paintress mcp` implements the MCP lifecycle (`initialize`, `notifications/initialized`, `tools/list`, `tools/call`) over stdio.
- `next_issue` reads completed issue ids, the next expedition number, and the latest PR from local projections.
- `update_gradient` persists gradient-changed events.
- `append_journal` persists expedition-completed events and writes journal / PR-index state.
- `dmail` emits report D-Mails through the transactional outbox — the only sanctioned emission path (refs issue 0031).
- The `/expedition-next` skill performs implementation, verification, PR creation, and report D-Mail composition from the claude-code session.

Ref: ADR 0017, ADR 0018, `internal/session/mcp_server.go`, `plugins/paintress/skills/expedition-next/SKILL.md`

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
