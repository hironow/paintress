# Architecture Decision Records

## Shared ADRs (canonical: phonewave)

0001-0005 are reserved. Canonical versions live in `phonewave/docs/adr/`.

| # | Decision | Linear |
|---|----------|--------|
| 0001 | cobra CLI framework adoption | MY-329 |
| 0002 | stdio convention (stdout=data, stderr=logs) | MY-339 |
| 0003 | OpenTelemetry noop-default + OTLP HTTP | MY-363 |
| 0004 | D-Mail Schema v1 specification | MY-352, MY-353 |
| 0005 | fsnotify daemon design | MY-363 |

## Extended Shared ADRs (S-series, canonical: phonewave)

Canonical versions live in phonewave `docs/adr/`. Referenced here for discoverability.

| # | Decision | Status |
|---|----------|--------|
| S0001 | ~~Logger as root package exception~~ | Superseded by S0005 |
| S0002 | JSONL append-only event sourcing pattern | Accepted |
| S0003 | Three-way approval contract | Accepted |
| S0004 | ~~Layer architecture conventions~~ | Superseded by S0005 |
| S0005 | Root infrastructure pattern and layer conventions | Accepted |
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
