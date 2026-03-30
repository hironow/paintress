package session

import (
	"context"
	"encoding/json"
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
	ToolName   string                    // CLI tool name for stream events (e.g. "paintress")
	StreamBus  port.SessionStreamPublisher // optional: live session event streaming
}

// Run executes the Claude CLI once with streaming, returning only the result text.
func (a *ClaudeAdapter) Run(ctx context.Context, prompt string, w io.Writer, opts ...port.RunOption) (string, error) {
	result, err := a.RunDetailed(ctx, prompt, w, opts...)
	return result.Text, err
}

// RunDetailed executes the Claude CLI once with streaming, returning the result
// text and provider session ID.
func (a *ClaudeAdapter) RunDetailed(ctx context.Context, prompt string, w io.Writer, opts ...port.RunOption) (port.RunResult, error) {
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
	if rc.ResumeSessionID != "" {
		args = append(args, "--resume", rc.ResumeSessionID)
	} else if rc.Continue {
		args = append(args, "--continue")
	}
	if len(rc.AllowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(rc.AllowedTools, ","))
	}
	args = append(args, "--verbose", "--output-format", "stream-json")
	// NOTE: --setting-sources "" skips settings loading but does NOT suppress CLAUDE.md auto-discovery.
	// --bare would suppress it but also disables OAuth. No individual flag exists to disable CLAUDE.md
	// discovery without disabling OAuth. Acceptable tradeoff: CLAUDE.md adds context but doesn't
	// cause context budget issues in practice.
	args = append(args, "--setting-sources", "") // Skip user/project settings (hooks, plugins, auto-memory) while preserving OAuth auth
	args = append(args, "--disable-slash-commands")

	// Settings and MCP config live under the tool's stateDir (e.g. .expedition/).
	// ConfigBase is the repo root (continent) where stateDir was initialized.
	// When ConfigBase is unset, fall back to WorkDir, then CWD.
	configBase := rc.ConfigBase
	if configBase == "" {
		configBase = effectiveDir(rc.WorkDir)
	}

	// Load tool-specific settings when available; warn if missing
	if settingsPath := ClaudeSettingsPath(configBase); ClaudeSettingsExists(configBase) {
		args = append(args, "--settings", settingsPath)
	} else if a.Logger != nil {
		a.Logger.Warn("Claude subprocess settings not found at %s", settingsPath)
		a.Logger.Warn("Run 'paintress mcp-config generate' to create settings.")
	}

	// Enforce MCP allowlist when .mcp.json (or legacy .run/mcp-config.json) exists
	if mcpPath := ResolveMCPConfigPath(configBase); mcpPath != "" {
		args = append(args, "--strict-mcp-config", "--mcp-config", mcpPath)
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
		return port.RunResult{}, fmt.Errorf("stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return port.RunResult{}, fmt.Errorf("claude start: %w", err)
	}

	var output strings.Builder
	var responseModel, responseID string
	var providerSessionID string
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

		if a.StreamBus != nil && a.ToolName != "" {
			normalizer := platform.NewStreamNormalizer(a.ToolName, domain.ProviderClaudeCode)
			emitter.SetStreamMessageHandler(func(msg *platform.StreamMessage, raw json.RawMessage) {
				if ev := normalizer.Normalize(msg, raw); ev != nil {
					a.StreamBus.Publish(ctx, *ev)
				}
			})
		}

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
			providerSessionID = result.SessionID
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
			return port.RunResult{Text: output.String(), ProviderSessionID: providerSessionID}, fmt.Errorf("claude exit: %w\n%s", waitErr, diagnostic)
		}
		return port.RunResult{Text: output.String(), ProviderSessionID: providerSessionID}, fmt.Errorf("claude exit: %w", waitErr)
	}
	select {
	case readError := <-streamErr:
		return port.RunResult{Text: output.String(), ProviderSessionID: providerSessionID}, fmt.Errorf("stream read: %w", readError)
	default:
	}
	return port.RunResult{Text: output.String(), ProviderSessionID: providerSessionID}, nil
}

// effectiveDir returns dir if non-empty, otherwise ".".
func effectiveDir(dir string) string {
	if dir != "" {
		return dir
	}
	return "."
}
