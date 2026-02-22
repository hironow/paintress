# 0009. Reserve Party Rate Limit Fallback

**Date:** 2026-02-23
**Status:** Accepted

## Context

paintress invokes Claude Code with a specified model (typically Opus) for each
expedition. The Anthropic API enforces rate limits that can temporarily block
requests. When rate-limited mid-session, the expedition times out and fails,
resetting the Gradient Gauge and stalling issue progress. Without automated
fallback, the operator must manually intervene or wait for the rate limit to
expire.

## Decision

Implement real-time rate limit signal detection with automatic model fallback:

1. **Signal detection** (`CheckOutput`): Scans the Claude process output stream
   for 7 text-based rate limit signals (`rate limit`, `rate_limit`,
   `too many requests`, `quota exceeded`, `usage limit`, `try again later`,
   `at capacity`) plus HTTP status code `429` (word-boundary matched via
   `hasHTTP429`). Detection is case-insensitive.

2. **Automatic switchover** (`onRateLimitDetected`): When a signal is detected
   and the active model is the primary, switches to `reserve[0]` (first
   reserve model). Sets a 30-minute cooldown timer. Increments hit counter
   for observability.

3. **Primary recovery** (`TryRecoverPrimary`): Called before each expedition.
   If the active model is a reserve and the cooldown has passed, switches
   back to the primary model.

4. **Manual force** (`ForceReserve`): Allows paintress to force-switch to
   reserve when a timeout is suspected to be rate-limit-related (e.g., the
   process hung without producing output).

5. **Prompt integration** (`FormatForPrompt`, `ActiveModel`): The active model
   is passed to `claude --model` at invocation time (`expedition.go`). The
   prompt includes reserve status so Claude Code is aware of potential quality
   differences.

## Consequences

### Positive
- Expeditions continue under rate limits at reduced model quality instead of
  failing entirely
- Primary model automatically recovers after cooldown without operator action
- Hit counter and status logging provide visibility into rate limit frequency

### Negative
- Signal detection is heuristic-based — false positives (e.g., output discussing
  rate limits) may trigger unnecessary fallback
- Reserve model quality degradation may produce lower-quality implementations,
  affecting Gradient Gauge progression
- 30-minute cooldown is a fixed heuristic that may not match actual API recovery
  time

### Neutral
- `ReserveParty` is thread-safe via `sync.RWMutex`, supporting concurrent
  `CheckOutput` calls from multiple expedition workers
- The reserve list is ordered by preference (`reserve[0]` is the first fallback)
