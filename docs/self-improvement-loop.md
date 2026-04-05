# paintress self-improvement loop

## Purpose

`paintress` is the implementation and repair side of the 4-tool loop.

It sits on the path:

`specification -> implementation -> verification -> correction`

and is responsible for turning corrective feedback into a concrete rerun, patch, or expedition result.

## What this tool now does

`paintress` now participates in the observable self-improvement loop by keeping corrective context attached to implementation reruns.

The current implementation does four things:

1. It matches incoming corrective feedback to the relevant expedition report.
2. It carries normalized corrective metadata into rerun-linked reports.
3. It keeps retry and escalation context visible on the next implementation result.
4. It stores provider pause state in coding session metadata using the shared provider-state vocabulary.

## Shared corrective metadata

The rerun path can carry metadata such as:

- `failure_type`
- `secondary_type`
- `target_agent`
- `recurrence_count`
- `corrective_action`
- `retry_allowed`
- `escalation_reason`
- `correlation_id`
- `trace_id`
- `outcome`

For `paintress`, this metadata is mainly used to keep an implementation correction thread attached to the next report sent back into the loop.

## Corrective rerun behavior

`paintress` does not decide the original diagnosis.

Instead, it preserves the corrective thread and emits the next report with enough context for `amadeus` to judge whether the rerun resolved the issue or failed again. Matching prefers:

1. wave target identity
2. issue ID fallback

This keeps implementation reruns inspectable instead of looking like unrelated new reports.

## Provider pause model

`paintress` uses the shared provider-state snapshot:

- `active`
- `waiting`
- `degraded`
- `paused`

Those states are persisted into coding session metadata together with:

- `provider_state`
- `provider_reason`
- `provider_retry_budget`
- `provider_resume_at`
- `provider_resume_when`

This keeps provider pause and expedition failure as separate concepts.

## Current scope

What is in:

- rerun correlation for corrective implementation feedback
- carry-forward of retry and escalation context
- provider pause state snapshots in session metadata

What is not in yet:

- learned patch generation policy
- long-horizon edit-strategy updates
- a standalone improvement controller

