package session

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ClaudeAdapter implements port.ClaudeRunner by executing the Claude CLI
// as a subprocess with CombinedOutput (non-streaming).
// For streaming use cases (Expedition.Run), use the existing inline logic.
type ClaudeAdapter struct {
	ClaudeCmd string
	Model     string
	Logger    domain.Logger
}

// Run executes the Claude CLI once. It writes output to w and returns the
// combined stdout+stderr as a string.
func (a *ClaudeAdapter) Run(ctx context.Context, prompt string, w io.Writer, opts ...port.RunOption) (string, error) {
	rc := port.ApplyOptions(opts...)

	model := a.Model
	_, span := platform.Tracer.Start(ctx, "claude.invoke",
		trace.WithAttributes(
			append([]attribute.KeyValue{
				attribute.String("claude.model", platform.SanitizeUTF8(model)),
			}, platform.GenAISpanAttrs(model)...)...,
		),
	)
	defer span.End()

	var args []string
	if model != "" {
		args = append(args, "--model", model)
	}
	if rc.Continue {
		args = append(args, "--continue")
	}
	if len(rc.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(rc.AllowedTools, ","))
	}
	args = append(args, "--dangerously-skip-permissions", "--print", "-p", prompt)

	cmd := platform.NewShellCmd(ctx, a.ClaudeCmd, args...)
	if rc.WorkDir != "" {
		cmd.Dir = rc.WorkDir
	}
	cmd.WaitDelay = 3 * time.Second

	out, err := cmd.CombinedOutput()
	output := string(out)

	if w != nil {
		_, _ = w.Write(out)
	}

	if err != nil {
		span.RecordError(err)
		return output, fmt.Errorf("claude exit: %w", err)
	}

	return output, nil
}
