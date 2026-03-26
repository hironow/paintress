package session

import (
	"context"
	"errors"
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
	if streamErr != nil {
		return output.String(), streamErr
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
