# Architecture Decision Records

## Numbering Scheme

| Range | Scope | Description |
|-------|-------|-------------|
| S0001-S0035 | Shared | Cross-tool decisions. All 4 tools follow these. |
| 0001+ (per tool) | Tool-specific | Each tool numbers its own ADRs independently. |

- **Shared ADRs** are maintained in `docs/shared-adr/` within each tool repository. All four tools keep identical copies.
- **Tool-specific ADRs** live in each tool's own `docs/adr/` with numbering starting at 0001.
- Semgrep rules enforcing shared ADRs are copied to each tool's `.semgrep/shared-adr.yaml`.

## Shared ADRs (see: [docs/shared-adr/](../shared-adr/))

| # | Decision | Status |
|---|----------|--------|
| S0001 | Cross-Tool Decision Index | Accepted |
| S0002 | cobra CLI framework adoption | Accepted |
| S0003 | stdio convention (stdout=data, stderr=logs) | Accepted |
| S0004 | OpenTelemetry noop-default + OTLP HTTP | Accepted |
| S0005 | D-Mail Schema v1 specification | Accepted |
| S0006 | fsnotify-based file watch daemon | Accepted |
| S0007 | Root infrastructure and layer conventions | Accepted |
| S0008 | cmd-eventsource import prohibition | Accepted |
| S0009 | SQLite WAL cooperative model for concurrent CLI | Accepted |
| S0010 | Reference data management pattern | Accepted |
| S0011 | COMMAND naming convention (imperative present tense) | Accepted |
| S0012 | POLICY pattern reference implementation | Accepted |
| S0013 | State directory naming convention | Accepted |
| S0014 | Root package file organization | Accepted |
| S0015 | Aggregate root and use case layer | Accepted |
| S0016 | Event Storming alignment and per-tool applicability | Accepted |
| S0017 | Data persistence boundaries (Linear/GitHub/local) | Accepted |
| S0018 | Accepted cross-tool divergence (default subcommand, storage model) | Accepted |
| S0019 | D-Mail receive-side validation (Postel's Law) | Accepted |
| S0020 | OTel Metrics Design | Accepted |
| S0021 | Cross-Tool Contract Testing | Accepted |
| S0022 | ~~CLI Argument Design Decisions~~ | Superseded by S0026 |
| S0023 | Event Delivery Guarantee Levels | Accepted |
| S0024 | Domain Model Maturity Assessment | Accepted |
| S0025 | RDRA Gap Resolution — D-Mail Protocol Extension | Accepted |
| S0026 | CLI Argument Design (Actual Implementation) | Accepted |
| S0027 | OTel env-file backend configuration | Accepted |
| S0028 | Usecase-adapter dependency inversion | Accepted |
| S0029 | Parse-don't-validate commands | Accepted |
| S0030 | Insight Data Persistence | Accepted |
| S0031 | D-Mail Context Extension | Accepted |
| S0032 | CVD-Friendly Signal Color Palette | Accepted |
| S0033 | Loop Safety Audit (2026-03) | Accepted |
| S0034 | Session-Usecase Boundary Clarification | Accepted |
| S0035 | D-Mail Wave Field Extension | Accepted |

## paintress-specific ADRs

| # | Decision |
|---|----------|
| [0001](0001-game-inspired-expedition-system.md) | Game-Inspired Expedition System |
| [0002](0002-per-worker-flag-checkpoint.md) | Per-Worker Flag Checkpoint |
| [0003](0003-archive-first-dmail-write.md) | Archive-First D-Mail Write |
| [0004](0004-reserve-party-rate-limit-fallback.md) | Reserve Party Rate Limit Fallback |
| [0005](0005-three-way-approval-contract.md) | Three-Way Approval Contract |
| [0006](0006-bounded-concurrency-pond-v2.md) | Bounded Concurrency Pond v2 |
| [0007](0007-defer-unlock-enforcement.md) | Defer-Unlock Enforcement |
| [0008](0008-root-package-layer-separation.md) | Root Package Layer Separation |
| [0009](0009-event-sourcing.md) | Event Sourcing |
| [0010](0010-event-sourcing-with-projections.md) | Event Sourcing with Projections |
| [0011](0011-usecase-adapter-dependency-inversion.md) | Usecase-Adapter Dependency Inversion |
| [0012](0012-parse-dont-validate-commands.md) | Parse-Don't-Validate Commands |
| [0013](0013-preflight-dmail-triage.md) | Preflight D-Mail Triage |
| [0015](0015-claude-cli-stream-json-requires-verbose.md) | Claude CLI stream-json Requires --verbose |
