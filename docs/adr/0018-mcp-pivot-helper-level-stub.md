# 0018. MCP pivot helper-level stub: extend enforcement to ClaudeAdapter / doctor probe / issues cobra

**Date:** 2026-05-22
**Status:** Accepted (extends [0017])

## Context

ADR 0017 (`MCP pivot: claude code session owns LLM, paintress Go CLI is
MCP server data plane`) pinned the architectural invariant
"LLM invocation is always human-initiated inside a claude code session"
and declared the semgrep gate `.semgrep/jun15-no-headless-llm.yaml`
as the mechanical enforcement.

The 2026-05-22 audit recorded in
`tap/refs/HTMLification/docs/issues/0028-jun15-pivot-transmigration-residue-cleanup.html`
discovered that the 0017 enforcement inventory listed only
`internal/session/expedition.Run()` as the legacy LLM invocation
loop, and stopped one layer above the actual helper. Three
production-reachable headless LLM paths survived the Phase 1
stub-out:

1. `internal/session/claude_adapter.go::ClaudeAdapter.RunDetailed`
   (≈280 lines) still built `[]string{..., "--print", ...}` and called
   `exec.Command(a.ClaudeCmd, args...)`. Reachable from
   `paintress issues` cobra → `usecase.FetchIssues` →
   `session.FetchIssuesViaMCP` → `runner.Run` →
   `ClaudeAdapter.RunDetailed`.
2. `internal/session/doctor.go::RunDoctor` ran a
   `newShellCmd(claudeCmd, "--print", "--verbose",
   "--output-format", "stream-json", "--max-turns", "1", "1+1=")`
   probe (the `claude-inference` check) every time a user invoked
   `paintress doctor`.
3. `internal/cmd/issues.go::runIssues` constructed the
   `ClaudeAdapter` and executed the chain above on every invocation,
   so any operator running `paintress issues` to inspect the Linear
   queue still spent credit-pool capacity.

The 0017 semgrep gate (rule `jun15-no-claude-print-exec-go`) used
`pattern-either: exec.Command("claude", "--print", ...)` literal
matching. Both surviving sites built args as a `[]string` slice and
spread them into `exec.Command(claudeCmd, args...)`, where neither
the binary path nor the flags were literal at the `exec.Command`
call site. The literal-only pattern missed the dynamic spread.

The audit found the same shape in dominator (`internal/session/k6_mcp_adapter.go`)
and produced refs/issues/0028 to track repo-wide cleanup.

## Decision

The Phase 1 enforcement is extended downward to every helper that
could synthesise a `claude --print` invocation, regardless of how
it is composed:

1. **`ClaudeAdapter.RunDetailed` is stub-replaced.** The 280-line
   body that built args, spawned the subprocess, streamed
   stream-json, and managed StreamBus publish/end is removed. The
   new body returns `port.RunResult{}, ErrMCPPivotDeprecated`
   (sentinel reused from `expedition.go`). Config fields are
   retained on the struct so existing composition roots compile.

2. **`doctor.go::RunDoctor` no longer runs the `claude-inference`
   probe.** Both `claude-inference` and the dependent
   `context-budget` check unconditionally report `CheckSkip` with
   message `"skipped (MCP pivot, refs/issues/0027); validate via
   claude code /expedition-next"`. The
   `checkClaudeInference` / `ExtractStreamResult` /
   `CheckContextBudget` helper functions remain (covered by
   `doctor_test.go`) so the pure-function unit tests stay green,
   but production code no longer calls them with a live probe.

3. **`paintress issues` cobra subcommand is fast-failed.** The
   subcommand now writes a deprecation notice to stderr and
   immediately returns `session.ErrMCPPivotDeprecated` before
   constructing the `ClaudeAdapter`, so no subprocess is spawned
   and no credit pool capacity is consumed. The
   `paintress.next_issue` MCP tool (advertised by
   `paintress mcp`) is the supported replacement.

4. **A new semgrep rule catches the dynamic-spread shape.** The
   rule `jun15-no-print-flag-literal-go` flags any `"--print"`
   string literal in production Go files under `internal/**` /
   `cmd/**` / `main.go`, with `_test.go` / `tests/**` / `.semgrep/**`
   excluded. This catches `args := []string{..., "--print", ...}`
   regardless of how `args` is later spread into `exec.Command`.

The new rule is applied symmetrically across all five tools
(sightjack / paintress / amadeus / phonewave / dominator) so that
future regressions are caught at the same gate regardless of which
repo introduces them.

## Enforcement inventory (refinement of 0017)

### Entry points (all now stub-protected)

- `internal/cmd/issues.go::runIssues` - fast-failed at the cobra
  level before any adapter construction.
- `internal/session/expedition.Run()` - stubbed in ADR 0017 Phase 1.
- `internal/session/claude_adapter.go::ClaudeAdapter.Run` /
  `ClaudeAdapter.RunDetailed` - stubbed in this ADR.
- `internal/session/doctor.go::RunDoctor` claude-inference probe -
  replaced with `CheckSkip` in this ADR.

### Bypass candidates (now blocked)

- Dynamic args spread into `exec.Command(claudeCmd, args...)`:
  blocked by `jun15-no-print-flag-literal-go`.
- Direct `exec.Command("claude", "--print", ...)` literal: still
  blocked by 0017's `jun15-no-claude-print-exec-go`.
- Shell wrappers: still blocked by 0017's
  `jun15-no-claude-print-shell-wrapper`.
- Anthropic SDK / `ANTHROPIC_API_KEY`: still blocked by 0017's
  `jun15-no-anthropic-*` rules.

### Tests proving coverage

- `internal/session/claude_adapter_test.go::TestClaudeAdapter_HasTimeoutSec`
  asserts the struct's config field survives the body removal.
- `internal/session/doctor_test.go::TestRunDoctor_MCPChecks_AllPassWithFakeClaude`
  asserts both `claude-inference` and `context-budget` report
  `CheckSkip` post pivot (positive coverage for the new behaviour).
- `internal/cmd/issues_test.go` asserts the `--state` flag and
  `--output` PersistentFlag still parse, so the cobra surface is
  preserved even though execution is fast-failed.
- `just semgrep` runs 76 rules with 0 findings, including the new
  `jun15-no-print-flag-literal-go` rule.

## Consequences

### Positive

- `paintress issues`, `paintress doctor`, and any caller that
  reaches `ClaudeAdapter.RunDetailed` now consume zero credit pool
  capacity post 2026-06-15.
- The enforcement gap that survived ADR 0017's Phase 1 (= "stop at
  the entry point, miss the helper") is closed by a regex-based
  rule that does not depend on flow-sensitive analysis.
- The new rule is repo-symmetric across all five tools, so a
  future maintainer who copies the same pattern into a different
  tool will hit the gate at PR time.

### Negative

- `paintress doctor` no longer verifies that the Claude binary
  can run a trivial inference. Operators must rely on
  `/expedition-next` inside claude code as the proof-of-life.
- The dynamic-spread regex (`"--print"`) is coarser than
  flow-sensitive analysis; it will flag legitimate documentation
  strings in production Go files. The exclude list
  (`_test.go` / `tests/**` / `.semgrep/**`) covers the known
  documentation locations; if a real false positive appears,
  refine the include path or move the literal into a constant
  that lives outside `internal/` / `cmd/`.

### Neutral

- Helper functions `checkClaudeInference`, `ExtractStreamResult`,
  `CheckContextBudget`, `WriteClaudeLog` remain in the package.
  Their unit tests (pure functions over fixture strings) still
  exercise them. Removing the call site does not justify removing
  the functions themselves.

## References

- [ADR 0017](0017-mcp-pivot.md) - the base architectural pin
  this ADR extends.
- `tap/refs/HTMLification/docs/issues/0028-jun15-pivot-transmigration-residue-cleanup.html`
  - 2026-05-22 audit report and the fix plan this ADR implements.
- `tap/refs/HTMLification/docs/archive/0027-jun15-mcp-pivot.html`
  - canonical multi-tool MCP pivot plan.
- `tap/refs/HTMLification/lessons/0027-jun15-mcp-pivot-post-mortem.html`
  - Pattern 09 ("stub the helpers, not just the entry points")
  documents the lesson behind this ADR.
- `.semgrep/jun15-no-headless-llm.yaml` - the gate that mechanically
  enforces this decision.
