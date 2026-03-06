# Architecture Decision Records

## Numbering Scheme

| Range | Scope | Description |
|-------|-------|-------------|
| 0000-0005 | Shared (canonical: phonewave) | Cross-tool decisions. All 4 tools follow these. |
| 0006+ (per tool) | Tool-specific | Each tool numbers its own ADRs starting from 0006. |
| S00XX | Shared additions (canonical: phonewave) | Post-initial shared decisions added during alignment. |

- **Shared ADRs (0000-0005)** live only in phonewave `docs/adr/`. Other tools reference them but do not copy them.
- **Tool-specific ADRs (0006+)** live in each tool's own `docs/adr/` with numbering starting at 0006.
- **S-series ADRs** are shared decisions added after the initial 0000-0005 set. They also live only in phonewave.
- Semgrep rules enforcing shared ADRs are copied to each tool's `.semgrep/shared-adr.yaml`.

## Shared ADRs (canonical: phonewave)

0000-0005 are reserved. Canonical versions live in `phonewave/docs/adr/`.

| # | Decision | Linear |
|---|----------|--------|
| 0001 | cobra CLI framework adoption | MY-329 |
| 0002 | stdio convention (stdout=data, stderr=logs) | MY-339 |
| 0003 | OpenTelemetry noop-default + OTLP HTTP | MY-363 |
| 0004 | D-Mail Schema v1 specification | MY-352, MY-353 |
| 0005 | fsnotify daemon design | MY-363 |

## S-series Shared ADRs (canonical: phonewave)

Canonical versions live in phonewave `docs/adr/`. Referenced here for discoverability.

| # | Decision | Status |
|---|----------|--------|
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
