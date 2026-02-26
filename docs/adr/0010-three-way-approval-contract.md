# 0010. Three-Way Approval Contract

**Date:** 2026-02-23
**Status:** Accepted — generalized in shared ADR S0003

## Context

When the D-Mail inbox contains HIGH severity messages, paintress must not
proceed with expeditions without human acknowledgment. However, the approval
mechanism can fail in two fundamentally different ways: the human intentionally
denies the request, or the approval infrastructure itself fails (e.g., the
external approval binary is not found, stdin is closed). These two failure
modes require different treatment — intentional denial is a clean exit, while
infrastructure failure should trigger a fail-closed abort.

## Decision

Adopt a three-way approval contract with session-level gating:

1. **Three-way return contract**: `Approver.RequestApproval(ctx, message)`
   returns `(approved bool, err error)`, encoding three distinct outcomes:
   - `(true, nil)` — Approved: expeditions proceed normally
   - `(false, nil)` — Denied: clean exit, no expeditions run (exit code 0)
   - `(false, err)` — Error: fail-closed abort, no expeditions run (exit code 1)

2. **Three implementations**:
   - `StdinApprover`: Prompts on terminal, reads `y`/`yes` for approval. Empty
     input or any other response is denial (safe default). Uses injected
     `io.Reader`/`io.Writer` for testability.
   - `CmdApprover`: Executes an external command via `sh -c`. Exit code 0 =
     approved, `*exec.ExitError` (non-zero exit) = denied, other errors
     (binary not found, permission denied) = technical failure propagated
     as error. Supports `{message}` placeholder with shell quoting.
   - `AutoApprover`: Always returns `(true, nil)`. Used with `--auto-approve`
     for unattended operation.

3. **Wiring priority** (`paintress.go`): `AutoApprove` > `ApproveCmd` >
   `StdinApprover`. For `StdinApprover`, the prompt output writer falls back
   to `os.Stderr` when the logger discards output (`io.Discard`) or when
   `DataOut` and logger share the same writer (would corrupt JSON output).

4. **Session-level gate**: The pre-flight HIGH severity check runs once
   before workers start (`Run()`). It does not re-run between expeditions.
   HIGH severity D-Mails arriving mid-expedition trigger `Notifier.Notify()`
   (fire-and-forget) but do not re-gate.

5. **Notify vs Approve error semantics**: `Notifier.Notify()` errors are logged
   as Warn and do not block expeditions. `Approver.RequestApproval()` errors
   are logged as Error and trigger fail-closed abort.

## Consequences

### Positive

- Fail-closed design — infrastructure failures default to the safe path (abort)
- Companion binary authors only need to follow exit code convention (0 = approve,
  non-zero = deny); the third path (execution error) is handled by paintress
- Session-level gate avoids concurrent `StdinApprover` reads with `--workers > 1`
- All three implementations are testable via the `Approver` interface

### Negative

- Session-level gate does not re-prompt for HIGH severity D-Mails arriving
  after initial approval (mitigated by notification via `Notifier`)
- `StdinApprover` blocks the entire session on a single goroutine read — no
  timeout unless the parent context has one

### Neutral

- `CmdApprover` uses `sh -c` for shell expansion, requiring proper escaping
  of the `{message}` placeholder (handled by `shellQuote`)
- The three-way contract is documented in `docs/approval-contract.md` with
  test coverage matrix
