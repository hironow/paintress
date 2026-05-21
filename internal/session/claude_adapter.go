package session

import (
	"context"
	"io"
	"os/exec"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// ClaudeAdapter implements port.ProviderRunner. Post jun15 MCP pivot
// (refs/issues/0027 Phase 1 + 0028 §4.2 residue cleanup), the Run /
// RunDetailed bodies return ErrMCPPivotDeprecated rather than executing
// `claude --print` via exec.Command. LLM inference now happens inside a
// human-initiated claude code interactive session driven by the
// paintress MCP server (`paintress mcp` subcommand) plus the
// /expedition-next slash command defined in
// plugins/paintress/skills/expedition-next/SKILL.md.
//
// The struct retains its config fields so call sites (issues / sweep /
// run) that construct it via ClaudeAdapter{...} still compile.
// Composition roots can wire it in but every Run / RunDetailed call
// short-circuits to ErrMCPPivotDeprecated until the MCP-driven pipeline
// replaces this adapter entirely.
//
// ErrMCPPivotDeprecated is declared in expedition.go and shared with
// Expedition.Run; callers can switch on errors.Is(err,
// ErrMCPPivotDeprecated) regardless of which entry point they hit.
type ClaudeAdapter struct { // nosemgrep: domain-primitives.public-string-field-go -- adapter config struct; ToolName is a config identifier [permanent]
	ClaudeCmd  string
	Model      string
	TimeoutSec int
	Logger     domain.Logger
	ToolName   string                      // CLI tool name for stream events (e.g. "paintress")
	StreamBus  port.SessionStreamPublisher // optional: live session event streaming
	// NewCmd retained on the struct so existing composition roots and
	// tests retain their wiring; the field is no longer read by Run /
	// RunDetailed post pivot.
	NewCmd func(ctx context.Context, name string, args ...string) *exec.Cmd
	// CancelFunc retained for the same reason as NewCmd above.
	CancelFunc func(cmd *exec.Cmd) func() error
}

// Run returns ErrMCPPivotDeprecated. The previous implementation
// invoked `claude --print` via exec.Command and streamed stream-json
// from stdout (~280 lines), which is forbidden post the jun15 MCP
// pivot (refs/issues/0027 §5 billing boundary, 0028 §4.2 residue
// cleanup).
func (a *ClaudeAdapter) Run(ctx context.Context, prompt string, w io.Writer, opts ...port.RunOption) (string, error) {
	result, err := a.RunDetailed(ctx, prompt, w, opts...)
	return result.Text, err
}

// RunDetailed returns ErrMCPPivotDeprecated. See Run for context.
func (a *ClaudeAdapter) RunDetailed(_ context.Context, _ string, _ io.Writer, _ ...port.RunOption) (port.RunResult, error) {
	if a.Logger != nil {
		a.Logger.Warn("paintress: ClaudeAdapter.RunDetailed() is deprecated (refs/issues/0027 jun15 MCP pivot, 0028 residue cleanup); use the claude code /expedition-next skill instead.")
	}
	return port.RunResult{}, ErrMCPPivotDeprecated
}
