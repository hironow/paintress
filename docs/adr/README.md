# Architecture Decision Records — paintress

## Shared ADRs (canonical: phonewave)

0001-0005 are reserved. Canonical versions live in `phonewave/docs/adr/`.

| # | Decision | Linear |
|---|----------|--------|
| 0001 | cobra CLI framework adoption | MY-329 |
| 0002 | stdio convention (stdout=data, stderr=logs) | MY-339 |
| 0003 | OpenTelemetry noop-default + OTLP HTTP | MY-363 |
| 0004 | D-Mail Schema v1 specification | MY-352, MY-353 |
| 0005 | fsnotify daemon design (phonewave-specific) | MY-363 |

## paintress-specific ADRs

| # | Decision | Linear |
|---|----------|--------|
| 0006 | [Game-inspired expedition system](0006-game-inspired-expedition-system.md) | MY-365 |
| 0007 | [Per-worker flag checkpoint with reconciliation](0007-per-worker-flag-checkpoint.md) | MY-362, MY-365 |
| 0008 | [Archive-first D-Mail write pattern](0008-archive-first-dmail-write.md) | MY-365 |
| 0009 | [Reserve Party rate limit fallback](0009-reserve-party-rate-limit-fallback.md) | MY-365 |
| 0010 | [Three-way approval contract](0010-three-way-approval-contract.md) | MY-323, MY-365 |
