package session

import (
	"context"
	"fmt"
	"io"
	"os"
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
	args = append(args, "--disable-slash-commands")
	if mcpPath := MCPConfigPath(effectiveDir(rc.WorkDir)); mcpPath != "" {
		if _, statErr := os.Stat(mcpPath); statErr == nil {
			args = append(args, "--strict-mcp-config", "--mcp-config", mcpPath)
		}
	}
	args = append(args, "--dangerously-skip-permissions", "--print", "-p", prompt)

	cmd := platform.NewShellCmd(ctx, a.ClaudeCmd, args...)
	if rc.WorkDir != "" {
		cmd.Dir = rc.WorkDir
	}
	cmd.WaitDelay = 3 * time.Second

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("claude start: %w", err)
	}

	var output strings.Builder
	var responseModel, responseID string
	streamErr := make(chan error, 1)
	done := make(chan struct{})

	go func() {
		defer close(done)
		sr := platform.NewStreamReader(stdout)
		if a.Logger != nil {
			sr.SetLogger(a.Logger)
		}
		emitter := platform.NewSpanEmittingStreamReader(sr, ctx, platform.Tracer)
		emitter.SetInput(prompt)
		result, messages, readErr := emitter.CollectAll()
		if readErr != nil {
			streamErr <- readErr
			return
		}

		for _, msg := range messages {
			switch msg.Type {
			case "assistant":
				text, _ := msg.ExtractText()
				if text != "" {
					if w != nil {
						_, _ = w.Write([]byte(text))
					}
					output.WriteString(text)
				}
				if am, _ := msg.ParseAssistantMessage(); am != nil {
					if am.Model != "" {
						responseModel = am.Model
					}
					if am.ID != "" {
						responseID = am.ID
					}
				}
			case "result":
				output.Reset()
				output.WriteString(msg.Result)
				span.SetAttributes(platform.GenAIResultAttrs(msg, responseModel, responseID)...)
			}
		}

		if rawEvents := emitter.RawEvents(); len(rawEvents) > 0 {
			span.SetAttributes(attribute.StringSlice("stream.raw_events", platform.SanitizeUTF8Slice(rawEvents)))
		}
		if result != nil && result.SessionID != "" {
			span.SetAttributes(platform.GenAISessionAttrs(result.SessionID)...)
		}
		if weaveAttrs := emitter.WeaveThreadAttrs(); len(weaveAttrs) > 0 {
			span.SetAttributes(weaveAttrs...)
		}
		if ioAttrs := emitter.WeaveIOAttrs(); len(ioAttrs) > 0 {
			span.SetAttributes(ioAttrs...)
		}
		if initAttrs := emitter.InitAttrs(); len(initAttrs) > 0 {
			span.SetAttributes(initAttrs...)
		}

		budget := platform.CalculateContextBudget(messages)
		span.SetAttributes(budget.Attrs()...)

		// Phase 5: persist raw events to .run/claude-logs/
		if rawEvents := emitter.RawEvents(); len(rawEvents) > 0 {
			if logErr := WriteClaudeLog(effectiveDir(rc.WorkDir), rawEvents); logErr != nil && a.Logger != nil {
				a.Logger.Warn("claude-log write: %v", logErr)
			}
		}
	}()

	<-done

	// Log captured stderr at debug level; suppress raw NDJSON from errors.
	if stderrBuf.Len() > 0 && a.Logger != nil {
		a.Logger.Debug("claude stderr:\n%s", stderrBuf.String())
	}

	if waitErr := cmd.Wait(); waitErr != nil {
		span.RecordError(waitErr)
		diagnostic := stderrBuf.String()
		if diagnostic != "" {
			if platform.IsNDJSON(diagnostic) {
				diagnostic = platform.SummarizeNDJSON(diagnostic)
			}
			return output.String(), fmt.Errorf("claude exit: %w\n%s", waitErr, diagnostic)
		}
		return output.String(), fmt.Errorf("claude exit: %w", waitErr)
	}
	select {
	case readError := <-streamErr:
		return output.String(), fmt.Errorf("stream read: %w", readError)
	default:
	}
	return output.String(), nil
}

// effectiveDir returns dir if non-empty, otherwise ".".
func effectiveDir(dir string) string {
	if dir != "" {
		return dir
	}
	return "."
}
