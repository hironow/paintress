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

| # | Decision |
|---|----------|
| S0001 | Logger as root package exception |
| S0002 | JSONL append-only event sourcing pattern |
| S0003 | Three-way approval contract |

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
