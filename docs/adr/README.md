# Architecture Decision Records

## Numbering Scheme

| Range | Scope | Description |
|-------|-------|-------------|
| 0000-0005 | Shared | Cross-tool decisions. All 4 tools follow these. |
| 0006+ (per tool) | Tool-specific | Each tool numbers its own ADRs starting from 0006. |
| S00XX | Shared additions | Post-initial shared decisions added during alignment. |

- **Shared ADRs** are maintained in `refs/shared-adr/` (tap parent repository). Individual tools reference them by ID but do not copy them.
- **Tool-specific ADRs (0006+)** live in each tool's own `docs/adr/` with numbering starting at 0006.
- Semgrep rules enforcing shared ADRs are copied to each tool's `.semgrep/shared-adr.yaml`.

## Shared ADRs (canonical: refs/shared-adr/)

| # | Decision |
|---|----------|
| 0000 | Cross-Tool Decision Index |
| 0001 | cobra CLI framework adoption |
| 0002 | stdio convention (stdout=data, stderr=logs) |
| 0003 | OpenTelemetry noop-default + OTLP HTTP |
| 0004 | D-Mail Schema v1 specification |
| 0005 | fsnotify-based file watch daemon |

## S-series Shared ADRs (canonical: refs/shared-adr/)

| # | Decision | Status |
|---|----------|--------|
| S0005 | Root infrastructure and layer conventions | Accepted |
| S0008 | cmd-eventsource import prohibition | Accepted |
| S0011 | SQLite WAL cooperative model for concurrent CLI | Accepted |
| S0012 | Reference data management pattern | Accepted |
| S0013 | COMMAND naming convention (imperative present tense) | Accepted |
| S0014 | POLICY pattern reference implementation | Accepted |
| S0015 | State directory naming convention | Accepted |
| S0016 | Root package file organization | Accepted |
| S0017 | Aggregate root and use case layer | Accepted |
| S0018 | Event Storming alignment and per-tool applicability | Accepted |
| S0019 | Data persistence boundaries (Linear/GitHub/local) | Accepted |
| S0020 | Accepted cross-tool divergence (default subcommand, storage model) | Accepted |
| S0021 | D-Mail receive-side validation (Postel's Law) | Accepted |
| S0022 | OTel Metrics Design | Accepted |
| S0023 | Cross-Tool Contract Testing | Accepted |
| S0024 | ~~CLI Argument Design Decisions~~ | Superseded by S0028 |
| S0025 | Event Delivery Guarantee Levels | Accepted |
| S0026 | Domain Model Maturity Assessment | Accepted |
| S0027 | RDRA Gap Resolution — D-Mail Protocol Extension | Accepted |
| S0028 | CLI Argument Design (Actual Implementation) | Accepted |
| S0029 | OTel env-file backend configuration | Accepted |
| S0030 | Usecase-adapter dependency inversion | Accepted |
| S0031 | Parse-don't-validate commands | Accepted |

## paintress-specific ADRs

| # | Decision | Linear |
|---|----------|--------|
| [0006](0006-game-inspired-expedition-system.md) | Game-inspired expedition system | MY-365 |
| [0007](0007-per-worker-flag-checkpoint.md) | Per-worker flag checkpoint with reconciliation | MY-362, MY-365 |
| [0008](0008-archive-first-dmail-write.md) | Archive-first D-Mail write pattern | MY-365 |
| [0009](0009-reserve-party-rate-limit-fallback.md) | Reserve Party rate limit fallback | MY-365 |
| [0010](0010-three-way-approval-contract.md) | Three-way approval contract | MY-323, MY-365 |
| [0011](0011-bounded-concurrency-pond-v2.md) | Bounded concurrency with pond/v2 | — |
| [0012](0012-defer-unlock-enforcement.md) | Defer-unlock enforcement via semgrep | — |
| [0013](0013-root-package-layer-separation.md) | Root package layer separation | — |
| [0014](0014-event-sourcing.md) | Event sourcing for state management | — |
