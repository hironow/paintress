# Rival Contract v1 (paintress — consumer)

paintress is the **consumer** of Rival Contract v1 specification D-Mails.
This document describes how a contract is parsed from the inbox and how
its sections feed the expedition prompt that drives implementation.

The full cross-tool plan lives at
[`refs/plans/2026-05-03-rival-contract-v1.md`](../../refs/plans/2026-05-03-rival-contract-v1.md).

## What it is

A Rival Contract v1 is the canonical Markdown body of a `kind: specification`
D-Mail produced by sightjack. paintress treats it as authoritative for
intent, ordered steps, boundaries, and verification evidence on the
assigned wave.

The contract supplements — it does NOT replace — the wave assignment.
paintress still drives off `wave.steps` for what to implement; the
contract clarifies why, in what order, and within which guardrails.

## Where the consumer lives

| Concern | File |
|---------|------|
| Parse Rival Contract from inbox D-Mail | `internal/harness/filter/dmail.go` |
| Pure parser + section accessors | `internal/harness/policy/rival_contract.go` |
| Expedition prompt (locale variants) | `internal/harness/filter/prompts/expedition_en.yaml` |
|  | `internal/harness/filter/prompts/expedition_ja.yaml` |
|  | `internal/harness/filter/prompts/expedition_fr.yaml` |

The consumer ingests the contract during expedition prompt construction.
No CLI verb was added in Phase 2.

## Section injection into the expedition prompt

The expedition prompt receives a focused `# Rival Contract: <title>`
block containing four sections:

- `## Intent` — verbatim from the contract
- `## Steps` — verbatim ordered list
- `## Boundaries` — deduplicated lines
- `## Evidence` — verbatim acceptance signals

`Domain` and `Decisions` are intentionally omitted from the prompt.
They are useful for humans and for amadeus drift checks, but the
implementing agent already has the wave assignment for scope and
should not be tempted to re-litigate prior decisions during
implementation.

The render function is `FormatRivalContractForPrompt` in
`internal/harness/policy/rival_contract.go`. It is pure, deterministic,
and contains no I/O.

## Boundary precedence rule

The expedition prompt enforces a hard rule:

> If the contract `## Boundaries` conflict with patterns inferred from
> the codebase, **prefer Boundaries**.

This rule is encoded directly in the locale prompt files (search for
`Contract Boundaries (Rival Contract v1)`). Without it, an agent might
silently override a boundary because the surrounding code shows
"another way." The contract is the source of truth for those guardrails.

## Wave target selection

The wave assignment continues to drive implementation scope:

- `wave.steps` defines the unit of work to deliver in this expedition.
- Contract `## Steps` provides the same ordered work in contract form;
  it must be consistent with `wave.steps` because the producer derives
  one from the other.
- Contract `## Evidence` supplements (does not replace) the existing
  acceptance signals from the wave.

The agent implements only the assigned wave step and uses the contract
to stay on-policy.

## Legacy fallback

A specification D-Mail with no `# Contract:` heading parses as
`ok=false`. paintress then falls back to the legacy expedition prompt
that uses the raw specification body. This is the migration path:
sightjack can roll out Rival Contract producers progressively without
breaking older waves still in flight.

## Cross-tool reference

| Tool | Role | Doc |
|------|------|-----|
| sightjack | producer | [sightjack/docs/rival-contract-v1.md](../../sightjack/docs/rival-contract-v1.md) |
| paintress | consumer (you are here) | this file |
| amadeus | drift controller | [amadeus/docs/rival-contract-v1.md](../../amadeus/docs/rival-contract-v1.md) |
| dominator | NFR judge | [dominator/docs/rival-contract-v1.md](../../dominator/docs/rival-contract-v1.md) |

## Required metadata seen on inbound contracts

paintress does not write contract metadata. It reads the producer's
output:

```yaml
metadata:
  contract_schema: rival-contract-v1
  contract_id: "<stable work-unit id>"
  contract_revision: "1"
  supersedes: ""
```

Only D-Mails with `contract_schema: rival-contract-v1` are routed
through the Rival Contract path. All other specification D-Mails take
the legacy path.

## Plan reference

- [`refs/plans/2026-05-03-rival-contract-v1.md`](../../refs/plans/2026-05-03-rival-contract-v1.md) — full design, phase plan, risks
- [`refs/scripts/check_rival_contract_docs.sh`](../../refs/scripts/check_rival_contract_docs.sh) — gap-check enforcement
