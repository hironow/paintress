# stdio Convention

Paintress follows the Unix convention of separating machine-readable data from human-readable diagnostics across standard streams.

## Stream Assignment

| Stream | Purpose | Implementation |
|--------|---------|----------------|
| **stdout** | Machine-readable output (JSON, expedition results) | `cmd.OutOrStdout()` |
| **stderr** | Human-readable progress, logs, errors | `cmd.ErrOrStderr()` |
| **stdin** | Prompt input to provider CLI subprocess only | `ProviderRunner.Run()` internal |

## Cobra Wiring

All cobra subcommands MUST use cobra's stream accessors:

```go
logger := platform.NewLogger(cmd.ErrOrStderr(), verbose)
```

Rules:

- Use `cmd.OutOrStdout()` for data output — never `os.Stdout` directly
- Use `cmd.ErrOrStderr()` for logs — never `os.Stderr` directly
- This enables cobra's `cmd.SetOut()` / `cmd.SetErr()` for testing

### Exceptions

Direct `os.Stderr` is acceptable only where cobra's `cmd` is unavailable:

| Location | Reason |
|----------|--------|
| `cmd/paintress/main.go` | Error handling after `root.ExecuteContext()` returns |
| `internal/tools/docgen/main.go` | Standalone tool outside cobra |
| `internal/session/paintress.go` | Quiet-mode fallback for interactive approval prompt (cmd layer passes `io.Discard` as errOut, but approval still needs visible output) |

## Pipeline Compatibility

The stream separation ensures correct behavior in Unix pipelines:

```bash
paintress status --json | jq '.expedition'    # stdout = JSON only
paintress status --json 2>/dev/null           # suppress stderr logs
paintress status --json 2>expedition.log      # split logs to file
```
