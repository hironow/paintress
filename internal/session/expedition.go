package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Expedition represents a single expedition into the Continent.
type Expedition struct {
	Number    int
	Continent string
	WorkDir   string // execution directory (worktree path or Continent)
	Config    domain.Config
	LogDir    string
	Logger    domain.Logger
	DataOut   io.Writer     // stdout-equivalent for streaming Claude output
	ErrOut    io.Writer     // stderr-equivalent for UI chrome output
	Notifier  port.Notifier // for mid-expedition HIGH severity notifications

	// Game mechanics
	Luminas     []domain.Lumina
	Gradient    *harness.GradientGauge
	Reserve     *harness.ReserveParty
	InboxDMails []domain.DMail // d-mails from inbox (for archiving after expedition)
	inboxOnce   sync.Once

	// Mid-expedition HIGH severity D-Mail tracking
	midHighMu    sync.Mutex
	midHighNames []string

	// Wave-centric mode: expedition target (step or wave)
	Target *domain.ExpeditionTarget

	// Resume context from a previous interrupted expedition (workers=0 only).
	// When non-empty, this is prepended to the expedition prompt so Claude
	// can continue from where the previous session left off.
	ResumeContext string

	// Parallel worker same-issue guard (nil in single-worker mode)
	ClaimRegistry *domain.IssueClaimRegistry

	// Mid-expedition issue-matched D-Mail routing (MY-361)
	currentIssueMu  sync.Mutex
	currentIssue    string
	midMatchedMu    sync.Mutex
	midMatchedMails []domain.DMail

	// capturedStderr holds the stderr output from the last Run() invocation.
	capturedStderr string

	// makeCmd overrides command creation for testing. If nil, exec.CommandContext is used.
	makeCmd func(ctx context.Context, name string, args ...string) *exec.Cmd

	// StreamBus receives live session stream events (optional, nil = no streaming).
	StreamBus port.SessionStreamPublisher
}

// Stderr returns the captured stderr output from the last Run() invocation.
func (e *Expedition) Stderr() string {
	return e.capturedStderr
}

// setCurrentIssue records the issue being worked on (called from watchFlag callback).
func (e *Expedition) setCurrentIssue(issue string) {
	e.currentIssueMu.Lock()
	defer e.currentIssueMu.Unlock()
	e.currentIssue = issue
}

// releaseClaim releases the issue claim held by this expedition, if any.
func (e *Expedition) releaseClaim() {
	if e.ClaimRegistry == nil {
		return
	}
	issue := e.getCurrentIssue()
	if issue != "" {
		e.ClaimRegistry.Release(issue)
	}
}

// getCurrentIssue returns the issue being worked on (thread-safe).
func (e *Expedition) getCurrentIssue() string {
	e.currentIssueMu.Lock()
	defer e.currentIssueMu.Unlock()
	return e.currentIssue
}

// MidMatchedDMails returns a copy of issue-matched D-Mails received mid-expedition.
func (e *Expedition) MidMatchedDMails() []domain.DMail {
	e.midMatchedMu.Lock()
	defer e.midMatchedMu.Unlock()
	if len(e.midMatchedMails) == 0 {
		return []domain.DMail{}
	}
	return append([]domain.DMail(nil), e.midMatchedMails...)
}

// appendMidHighName appends a HIGH severity D-Mail name (thread-safe).
func (e *Expedition) appendMidHighName(name string) {
	e.midHighMu.Lock()
	defer e.midHighMu.Unlock()
	e.midHighNames = append(e.midHighNames, name)
}

// appendMidMatchedMail appends an issue-matched D-Mail (thread-safe).
func (e *Expedition) appendMidMatchedMail(dm domain.DMail) {
	e.midMatchedMu.Lock()
	defer e.midMatchedMu.Unlock()
	e.midMatchedMails = append(e.midMatchedMails, dm)
}

// MidHighSeverityDMails returns names of HIGH severity D-Mails received mid-expedition.
func (e *Expedition) MidHighSeverityDMails() []string {
	e.midHighMu.Lock()
	defer e.midHighMu.Unlock()
	return append([]string(nil), e.midHighNames...)
}

// BuildPrompt generates the expedition prompt in the configured language.
func (e *Expedition) BuildPrompt() string {
	projCfg, err := LoadProjectConfig(e.Continent)
	if err != nil {
		e.Logger.Warn("project config load failed: %v", err)
		projCfg = &domain.ProjectConfig{}
	}

	data := domain.PromptData{
		Number:          e.Number,
		Timestamp:       time.Now().Format("2006-01-02 15:04:05"),
		Bt:              "`",
		Cb:              "```",
		LuminaSection:   harness.FormatLuminaForPrompt(e.Luminas),
		GradientSection: e.Gradient.FormatForPrompt(),
		ReserveSection:  e.Reserve.FormatForPrompt(),
		BaseBranch:      e.Config.BaseBranch,
		DevURL:          e.Config.DevURL,
		ContextSection:  e.loadContextSection() + e.resumeSection(),
		InboxSection:    e.loadInboxSection(),
		LinearTeam:      projCfg.TrackerTeam(),
		LinearProject:   projCfg.TrackerProject(),
		MissionSection:  harness.MissionText(harness.MustDefaultPromptRegistry(), domain.Lang, e.Target != nil),
		WaveTarget:      e.Target,
	}

	return harness.RenderExpeditionPrompt(harness.MustDefaultPromptRegistry(), domain.Lang, data)
}

func (e *Expedition) loadInboxSection() string {
	e.inboxOnce.Do(func() {
		if len(e.InboxDMails) > 0 {
			return // already loaded externally (e.g., by HIGH severity gate)
		}
		dmails, err := ScanInbox(context.Background(), e.Continent)
		if err != nil {
			e.Logger.Warn("inbox scan failed: %v", err)
			return
		}
		e.InboxDMails = dmails
	})
	return harness.FormatDMailForPrompt(e.InboxDMails)
}

func (e *Expedition) loadContextSection() string {
	ctx, err := readContextFilesWithLogger(e.Continent, e.Logger)
	if err != nil {
		e.Logger.Warn("context injection failed: %v", err)
		return ""
	}
	return ctx
}

func (e *Expedition) resumeSection() string {
	if e.ResumeContext == "" {
		return ""
	}
	return "\n\n## Previous Session Context (Resume)\n\n" + e.ResumeContext
}

// Run executes the expedition with timeout and streaming output.
func (e *Expedition) Run(ctx context.Context) (string, error) {
	if e.DataOut == nil {
		e.DataOut = io.Discard
	}
	if e.ErrOut == nil {
		e.ErrOut = io.Discard
	}
	promptStart := time.Now()
	prompt := e.BuildPrompt()
	promptBuildDuration := time.Since(promptStart)

	promptFile := filepath.Join(e.LogDir, fmt.Sprintf("expedition-%03d-prompt.md", e.Number))
	if err := os.WriteFile(promptFile, []byte(prompt), 0644); err != nil {
		return "", fmt.Errorf("failed to write prompt: %w", err)
	}

	timeout := time.Duration(e.Config.TimeoutSec) * time.Second
	expCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	model := e.Reserve.ActiveModel()

	expCtx, invokeSpan := platform.Tracer.Start(expCtx, "claude.invoke",
		trace.WithAttributes(
			append([]attribute.KeyValue{
				attribute.String("claude.model", platform.SanitizeUTF8(model)),
				attribute.Int("expedition.number", e.Number),
				attribute.Int("claude.timeout_sec", e.Config.TimeoutSec),
			}, platform.GenAISpanAttrs(model)...)...,
		),
	)
	defer invokeSpan.End()

	// Record prompt build duration as span attribute for telemetry breakdown.
	breakdown := domain.ExpeditionDurationBreakdown{
		PromptBuildDuration: promptBuildDuration,
	}
	invokeSpan.SetAttributes(breakdown.SpanAttributes()...)

	claudeCmd := e.Config.ClaudeCmd

	newCmd := e.makeCmd
	if newCmd == nil {
		newCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return platform.NewShellCmd(ctx, name, args...)
		}
	}

	cmd := newCmd(expCtx, claudeCmd,
		"--model", model,
		"--verbose",
		"--output-format", "stream-json",
		"--dangerously-skip-permissions",
		"--print",
		"-p", prompt,
	)
	workDir := e.WorkDir
	if workDir == "" {
		workDir = e.Continent
	}
	cmd.Dir = workDir

	if err := os.MkdirAll(filepath.Join(workDir, domain.StateDir, ".run"), 0755); err != nil {
		return "", fmt.Errorf("create expedition run dir: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe failed: %w", err)
	}
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	outputFile := filepath.Join(e.LogDir, fmt.Sprintf("expedition-%03d-output.txt", e.Number))
	outFile, err := os.Create(outputFile)
	if err != nil {
		return "", fmt.Errorf("output file creation failed: %w", err)
	}
	defer outFile.Close()

	// Clear stale current_issue from flag.md before starting the process.
	if stale := ReadFlag(workDir); stale.CurrentIssue != "" {
		WriteFlag(workDir, stale.LastExpedition, stale.LastIssue, stale.LastStatus, stale.Remaining, stale.MidHighSeverity)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s start failed: %w", claudeCmd, err)
	}

	// Start flag.md watcher to detect issue selection in real-time
	watchCtx, watchCancel := context.WithCancel(expCtx)
	defer watchCancel()
	go watchFlag(watchCtx, workDir, e.Logger, func(issue, title string) {
		if e.ClaimRegistry != nil {
			// Release old claim before attempting new one (prevents claim leak on issue switch)
			if old := e.getCurrentIssue(); old != "" && old != issue {
				e.ClaimRegistry.Release(old)
			}
			ok, holder := e.ClaimRegistry.TryClaim(issue, e.Number)
			if !ok {
				e.Logger.Warn("Expedition #%d: issue %s already claimed by expedition #%d — cancelling expedition", e.Number, issue, holder)
				cancel() // cancel the expedition context to kill the Claude process
				return
			}
		}
		e.setCurrentIssue(issue)
		invokeSpan.AddEvent("issue.picked",
			trace.WithAttributes(
				attribute.String("issue_id", platform.SanitizeUTF8(issue)),
				attribute.String("issue_title", platform.SanitizeUTF8(title)),
			),
		)
		e.Logger.Info("Expedition #%d: issue picked — %s (%s)", e.Number, issue, title)
	}, nil)

	// Start inbox watcher to log d-mails arriving mid-expedition.
	seenFiles := make(map[string]bool)
	for _, dm := range e.InboxDMails {
		seenFiles[dm.Name] = true
	}
	inboxDone := make(chan struct{})
	go func() {
		defer close(inboxDone)
		insightWriter := NewInsightWriter(domain.InsightsDir(e.Continent), domain.RunDir(e.Continent))
		watchInbox(watchCtx, e.Continent, func(dm domain.DMail) {
			if seenFiles[dm.Name] {
				return
			}
			seenFiles[dm.Name] = true
			WriteCorrectionInsight(insightWriter, dm, e.Logger)
			domain.LogBanner(e.Logger, domain.BannerRecv, string(dm.Kind), dm.Name, dm.Description)
			if dm.Severity == "high" {
				e.appendMidHighName(dm.Name)
				e.Logger.Warn("HIGH severity d-mail received mid-expedition: %s", dm.Name)
				if e.Notifier != nil {
					_ = e.Notifier.Notify(watchCtx, "Paintress", fmt.Sprintf("HIGH severity D-Mail mid-expedition: %s", dm.Name))
				}
			} else {
				e.Logger.Info("Expedition #%d: d-mail received — %s (%s)", e.Number, dm.Name, dm.Kind)
			}
			if cur := e.getCurrentIssue(); cur != "" && domain.ContainsIssue(dm.Issues, cur) {
				e.appendMidMatchedMail(dm)
				e.Logger.Info("Expedition #%d: d-mail routed to current issue %s — %s", e.Number, cur, dm.Name)
			}
		}, nil)
	}()

	// Streaming goroutine: tee to terminal + file + buffer + rate limit detection
	var output strings.Builder
	var responseModel, responseID string
	streamErr := make(chan error, 1)
	done := make(chan struct{})
	// expNormalizer is declared here (outside goroutine) so the defer can call SessionEnd().
	var expNormalizer *platform.StreamNormalizer
	defer func() {
		if expNormalizer != nil && e.StreamBus != nil {
			endEv := expNormalizer.SessionEnd("", nil)
			e.StreamBus.Publish(context.Background(), endEv)
		}
	}()

	go func() {
		defer close(done)
		sr := platform.NewStreamReader(stdout)
		streamDest := e.DataOut
		if e.Config.OutputFormat == "json" {
			streamDest = nil
		}
		var writer io.Writer
		if streamDest != nil {
			writer = io.MultiWriter(streamDest, outFile)
		} else {
			writer = outFile
		}

		emitter := platform.NewSpanEmittingStreamReader(sr, expCtx, platform.Tracer)
		emitter.SetInput(prompt)

		// Wire live stream event bus for expedition path.
		// expNormalizer is declared outside the handler so SessionEnd() can access
		// saved usage data after CollectAll completes.
		if e.StreamBus != nil {
			expNormalizer = platform.NewStreamNormalizer("paintress", domain.ProviderClaudeCode)
			emitter.SetStreamMessageHandler(func(msg *platform.StreamMessage, raw json.RawMessage) {
				if ev := expNormalizer.Normalize(msg, raw); ev != nil {
					e.StreamBus.Publish(expCtx, *ev)
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
					writer.Write([]byte(text))
					output.WriteString(text)
				}
				if e.Reserve.CheckOutput(text) {
					invokeSpan.AddEvent("rate_limit.detected")
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
				if msg.Result != "" {
					output.Reset()
					output.WriteString(msg.Result)
				}
				invokeSpan.SetAttributes(platform.GenAIResultAttrs(msg, responseModel, responseID)...)
			}
		}

		// Attach raw events and session ID to the invoke span
		if rawEvents := emitter.RawEvents(); len(rawEvents) > 0 {
			invokeSpan.SetAttributes(attribute.StringSlice("stream.raw_events", platform.SanitizeUTF8Slice(rawEvents)))
		}
		if result != nil && result.SessionID != "" {
			invokeSpan.SetAttributes(platform.GenAISessionAttrs(result.SessionID)...)
		}

		// Weave thread attributes for trace organization
		if weaveAttrs := emitter.WeaveThreadAttrs(); len(weaveAttrs) > 0 {
			invokeSpan.SetAttributes(weaveAttrs...)
		}
		if ioAttrs := emitter.WeaveIOAttrs(); len(ioAttrs) > 0 {
			invokeSpan.SetAttributes(ioAttrs...)
		}
		if initAttrs := emitter.InitAttrs(); len(initAttrs) > 0 {
			invokeSpan.SetAttributes(initAttrs...)
		}

		// Context budget measurement
		budget := platform.CalculateContextBudget(messages)
		invokeSpan.SetAttributes(budget.Attrs()...)
		if warning := budget.WarningMessage(platform.DefaultContextBudgetThreshold); warning != "" {
			e.Logger.Warn("%s", warning)
		}
	}()

	<-done

	var readError error
	select {
	case sErr := <-streamErr:
		readError = sErr
	default:
	}

	watchCancel()
	<-inboxDone

	err = cmd.Wait()
	e.capturedStderr = stderrBuf.String()
	if e.Config.OutputFormat != "json" {
		fmt.Fprintln(e.ErrOut)
	}

	if expCtx.Err() == context.DeadlineExceeded {
		invokeSpan.AddEvent("expedition.timeout",
			trace.WithAttributes(attribute.String("timeout", timeout.String())), // nosemgrep: otel-attribute-string-unsanitized -- time.Duration.String() always produces valid UTF-8 [permanent]
		)
		invokeSpan.AddEvent("expedition.failed",
			trace.WithAttributes(attribute.String("failure_type", "timeout")),
		)
		return output.String(), fmt.Errorf("timeout after %v", timeout)
	}
	if ctx.Err() == context.Canceled {
		invokeSpan.AddEvent("expedition.failed",
			trace.WithAttributes(attribute.String("failure_type", "interrupted")),
		)
		return output.String(), fmt.Errorf("interrupted")
	}

	if readError != nil {
		invokeSpan.AddEvent("expedition.failed",
			trace.WithAttributes(attribute.String("failure_type", "stream_error")),
		)
		return output.String(), fmt.Errorf("stream read: %w", readError)
	}

	if err != nil {
		invokeSpan.AddEvent("expedition.failed",
			trace.WithAttributes(attribute.String("failure_type", "execution_error")),
		)
	} else {
		invokeSpan.AddEvent("expedition.succeeded")
	}

	return output.String(), err
}

// MaxContextFileBytes is the maximum allowed size for a single context file.
// Files exceeding this limit are skipped with a warning log.
const MaxContextFileBytes int64 = 512 * 1024 // 512 KiB

// MaxContextTotalBytes is the maximum total size of all context files injected
// into the prompt. Loading stops early once this limit is reached.
const MaxContextTotalBytes int64 = 2 * 1024 * 1024 // 2 MiB

// ReadContextFiles reads all .md files from .expedition/context/ and
// concatenates them into a single string for prompt injection.
// Files that exceed MaxContextFileBytes are skipped with a warning log.
// Loading stops when the accumulated total would exceed MaxContextTotalBytes.
func ReadContextFiles(continent string) (string, error) {
	return readContextFilesWithLogger(continent, nil)
}

// readContextFilesWithLogger is the internal implementation that accepts an
// optional logger for testing. When logger is nil, the function is a no-op
// for warnings (callers should pass a real logger via loadContextSection).
func readContextFilesWithLogger(continent string, logger domain.Logger) (string, error) {
	dir := domain.ContextDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("reading context directory: %w", err)
	}

	var sortedEntries []os.DirEntry
	sortedEntries = append(sortedEntries, entries...)
	// Sort by name for deterministic output
	for i := range sortedEntries {
		for j := i + 1; j < len(sortedEntries); j++ {
			if sortedEntries[i].Name() > sortedEntries[j].Name() {
				sortedEntries[i], sortedEntries[j] = sortedEntries[j], sortedEntries[i]
			}
		}
	}

	var buf strings.Builder
	var totalBytes int64
	for _, e := range sortedEntries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}

		// Per-file size guard: skip files that exceed the per-file limit.
		info, err := e.Info()
		if err != nil {
			return "", fmt.Errorf("stat context file %s: %w", e.Name(), err)
		}
		if info.Size() > MaxContextFileBytes {
			if logger != nil {
				logger.Warn("context file %s exceeds size limit (%d bytes > %d), skipping", e.Name(), info.Size(), MaxContextFileBytes)
			}
			continue
		}

		// Total size guard: stop loading once the accumulator would exceed the total limit.
		if totalBytes+info.Size() > MaxContextTotalBytes {
			break
		}

		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", fmt.Errorf("reading context file %s: %w", e.Name(), err)
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		buf.WriteString(fmt.Sprintf("### %s\n\n", name))
		buf.Write(content)
		buf.WriteString("\n\n")
		totalBytes += info.Size()
	}
	return buf.String(), nil
}
