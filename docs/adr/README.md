# Paintress Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the paintress tool.

## Cross-Tool ADR Management

The phonewave ecosystem uses **Option C (hybrid)** for ADR management (MY-363):

- **phonewave** holds the canonical version of shared ADRs (0001-0004)
- Each tool maintains its own `docs/adr/` with independent numbering
- Cross-references use Linear issue numbers (MY-xxx) as stable identifiers

## Shared ADRs (canonical in phonewave)

Numbers 0001-0005 are reserved for cross-tool alignment with phonewave's canonical ADRs.

| # | Decision | Linear |
|---|----------|--------|
| 0001 | cobra CLI framework adoption | MY-329 |
| 0002 | stdio convention (stdout=data, stderr=logs) | MY-339 |
| 0003 | OpenTelemetry noop-default + OTLP HTTP | MY-363 |
| 0004 | D-Mail Schema v1 specification | MY-352, MY-353 |
| 0005 | *(phonewave-specific: fsnotify daemon)* | MY-363 |

## Paintress-Specific ADRs

| # | Decision | Linear |
|---|----------|--------|
| [0006](0006-game-inspired-expedition-system.md) | Game-inspired expedition system | MY-365 |
| [0007](0007-per-worker-flag-checkpoint.md) | Per-worker flag checkpoint with reconciliation | MY-362, MY-365 |
| [0008](0008-archive-first-dmail-write.md) | Archive-first D-Mail write pattern | MY-365 |
| [0009](0009-reserve-party-rate-limit-fallback.md) | Reserve Party rate limit fallback | MY-365 |
| [0010](0010-three-way-approval-contract.md) | Three-way approval contract | MY-323, MY-365 |
