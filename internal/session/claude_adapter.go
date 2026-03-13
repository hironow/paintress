package session

import (
	"context"
	"errors"
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
// as a subprocess with streaming (--output-format stream-json).
type ClaudeAdapter struct {
	ClaudeCmd  string
	Model      string
	TimeoutSec int
	Logger     domain.Logger
}

// Run executes the Claude CLI once with streaming. It writes assistant text
// to w incrementally and returns the result text (or concatenated assistant
// text if no result message appears).
func (a *ClaudeAdapter) Run(ctx context.Context, prompt string, w io.Writer, opts ...port.RunOption) (string, error) {
	rc := port.ApplyOptions(opts...)

	model := a.Model
	if rc.Model != "" {
		model = rc.Model
	}
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
	args = append(args, "--verbose", "--output-format", "stream-json")
	args = append(args, "--dangerously-skip-permissions", "--print", "-p", prompt)

	cmd := platform.NewShellCmd(ctx, a.ClaudeCmd, args...)
	if rc.WorkDir != "" {
		cmd.Dir = rc.WorkDir
	}
	cmd.WaitDelay = 3 * time.Second

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("claude start: %w", err)
	}

	reader := platform.NewStreamReader(stdout)
	reader.SetLogger(a.Logger)

	var output strings.Builder
	var streamErr error
	for {
		msg, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			streamErr = fmt.Errorf("stream read: %w", err)
			break
		}
		if msg == nil {
			continue
		}
		if msg.Type == "assistant" {
			text, extractErr := msg.ExtractText()
			if extractErr == nil && text != "" {
				output.WriteString(text)
				if w != nil {
					_, _ = w.Write([]byte(text))
				}
			}
		}
		if msg.Type == "result" {
			output.Reset()
			output.WriteString(msg.Result)
		}
	}

	if waitErr := cmd.Wait(); waitErr != nil {
		span.RecordError(waitErr)
		return output.String(), fmt.Errorf("claude exit: %w", waitErr)
	}
	if streamErr != nil {
		return output.String(), streamErr
	}
	return output.String(), nil
}
