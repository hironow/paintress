# 0013. Pre-Flight D-Mail Triage

**Date:** 2026-03-09
**Status:** Accepted

## Context

D-Mails arriving in paintress's inbox can carry an `action` field (retry, escalate, resolve). The existing `handleFeedbackAction` only processes actions on mid-expedition issue-matched D-Mails (MY-361). D-Mails without matching issue IDs were passed unprocessed to the expedition prompt, meaning their action fields were silently ignored.

This created a gap: an `action=escalate` D-Mail from amadeus would never trigger escalation if it didn't match the current expedition's issue.

## Decision

Add a `triagePreFlightDMails` step between inbox scan and expedition creation in `runWorker`. This processes ALL inbox D-Mails' action fields before the expedition starts:

- `escalate`: handled immediately via `handleEscalation`, archived, removed from expedition
- `resolve`: logged, archived, removed from expedition
- `retry` with issues: retry-tracked via `RetryTracker`; if count exceeds `MaxRetries`, escalated and removed; otherwise kept for expedition
- `retry` without issues: kept for expedition (no issue key to track)
- no action / unknown action: passed through to expedition prompt unchanged

Triaged-out D-Mails are archived immediately via `ArchiveInboxDMail`. Pass-through D-Mails are archived after expedition completion (existing behavior).

## Consequences

### Positive
- All D-Mail action fields are processed regardless of issue matching
- Escalation path works for cross-tool feedback (amadeus -> paintress)
- Retry counting works for repeated failures without mid-expedition match
- Consistent with Postel's law (S0021): unknown actions pass through

### Negative
- Two code paths for action processing: pre-flight (all D-Mails) and post-expedition (mid-matched only)
- Pre-flight triage adds latency before expedition start (negligible: O(n) loop over inbox)

### Neutral
- Mid-expedition D-Mail handling (watchInbox + handleFeedbackAction) unchanged
- D-Mails without action continue to be included in expedition prompts as before
