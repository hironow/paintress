# Approval Contract

The pre-flight HIGH severity gate uses a three-way approval contract. `Approver.RequestApproval()` returns `(approved bool, err error)`, producing three distinct outcomes that paintress handles differently.

## Three-Way Contract

```
+---------------------+
|  RequestApproval()  |
+---------------------+
         |
    +----+----+----+
    |         |         |
approved  denied     error
(true,nil) (false,nil) (false,err)
    |         |         |
 continue   abort     abort
 exit 0    exit 0    exit 1
```

Legend:
- approved: Expedition continues normally
- denied: Clean exit, no expeditions run
- error: Fail-closed abort, no expeditions run

| Outcome | `approved` | `err` | paintress exit code | Log level |
|---------|-----------|-------|---------------------|-----------|
| Approved | `true` | `nil` | 0 (expeditions run) | - |
| Denied | `false` | `nil` | 0 (clean exit) | `Warn` |
| Error | `false` | non-nil | 1 (fail-closed) | `Error` |

## Gate Scope: Session-Level

The pre-flight gate runs **once** before workers start. It does not re-run between expeditions.

```
Run()
  |
  +-- ScanInbox (pre-flight)
  +-- FilterHighSeverity
  +-- [if HIGH] Notify + RequestApproval  <-- gate (once)
  |
  +-- runWorker (loop)
        |
        +-- ScanInbox (per-expedition, for prompt data)
        +-- expedition.Run()
        |     +-- inbox_watcher (fsnotify)
        |           +-- [if HIGH] Notify  <-- notification only, no gate
        +-- (next expedition...)
```

Legend:
- gate: Blocking approval request (session-scoped)
- notification: Fire-and-forget alert (per-event)

This is intentional for two reasons:

1. **Concurrent safety**: Per-expedition gating with `--workers > 1` would cause concurrent `StdinApprover` reads. Serializing with a mutex would block workers against each other.
2. **Session semantics**: The gate answers "should this session proceed?" Once approved, the session continues. New HIGH severity d-mails arriving mid-run trigger notifications via `inbox_watcher` (fsnotify), but do not re-gate.

## CmdApprover: ExitError vs Execution Error

`CmdApprover` executes an external command via `sh -c`. Two types of failures map to different contract outcomes:

- **`*exec.ExitError`** (process started, exited non-zero): Treated as intentional denial. Returns `(false, nil)`.
- **Other errors** (binary not found, permission denied): Treated as technical failure. Returns `(false, err)`.

The distinction uses `errors.As(err, &exitErr)` where `exitErr` is `*exec.ExitError`.

## Companion Binary Implications

Companion binaries (`paintress-tg`, `paintress-discord`, `paintress-slack`) only need to follow the exit code contract:

- Exit 0 = approved
- Exit non-zero = denied

The third path (execution error) is handled by paintress when the binary itself cannot be started. Companion binary authors do not need to handle this case.

## Notify vs Approve: Error Semantics

Notification and approval have different error handling:

| Interface | Error behavior |
|-----------|---------------|
| `Notifier.Notify()` | Fire-and-forget. Errors logged as `Warn`, expedition continues. |
| `Approver.RequestApproval()` | Fail-closed. Errors logged as `Error`, paintress exits 1. |

## Test Coverage

| Contract | Test |
|----------|------|
| exit 0 = approved | `TestCmdApprover_ExitZero` |
| non-zero exit = denied | `TestCmdApprover_ExitNonZero` |
| execution error = error propagation | `TestCmdApprover_ExecutionError_SurfacesError` |
| approval error = fail-closed (exit 1) | `TestHighSeverityGate_ApprovalError_FailsClosed` |
| scan error = fail-closed (exit 1) | `TestHighSeverityGate_ScanError_FailsClosed` |
| `{message}` placeholder expansion | `TestCmdApprover_PlaceholderReplacement` |
| shell metacharacter escaping | `TestCmdApprover_EscapesShellMetacharacters` |
