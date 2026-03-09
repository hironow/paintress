package session

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hironow/paintress/internal/domain"
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
	Gradient    *domain.GradientGauge
	Reserve     *domain.ReserveParty
	InboxDMails []domain.DMail // d-mails from inbox (for archiving after expedition)
	inboxOnce   sync.Once

	// Mid-expedition HIGH severity D-Mail tracking
	midHighMu    sync.Mutex
	midHighNames []string

	// Mid-expedition issue-matched D-Mail routing (MY-361)
	currentIssueMu  sync.Mutex
	currentIssue    string
	midMatchedMu    sync.Mutex
	midMatchedMails []domain.DMail

	// makeCmd overrides command creation for testing. If nil, exec.CommandContext is used.
	makeCmd func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// errWriter returns ErrOut or io.Discard if nil (nil-safe accessor for tests).
func (e *Expedition) errWriter() io.Writer {
	if e.ErrOut != nil {
		return e.ErrOut
	}
	return io.Discard
}

// setCurrentIssue records the issue being worked on (called from watchFlag callback).
func (e *Expedition) setCurrentIssue(issue string) {
	e.currentIssueMu.Lock()
	defer e.currentIssueMu.Unlock()
	e.currentIssue = issue
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
		LuminaSection:   domain.FormatLuminaForPrompt(e.Luminas),
		GradientSection: e.Gradient.FormatForPrompt(),
		ReserveSection:  e.Reserve.FormatForPrompt(),
		BaseBranch:      e.Config.BaseBranch,
		DevURL:          e.Config.DevURL,
		ContextSection:  e.loadContextSection(),
		InboxSection:    e.loadInboxSection(),
		LinearTeam:      projCfg.TrackerTeam(),
		LinearProject:   projCfg.TrackerProject(),
		MissionSection:  platform.MissionText(domain.Lang),
	}

	return platform.RenderExpeditionPrompt(domain.Lang, data)
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
	return domain.FormatDMailForPrompt(e.InboxDMails)
}

func (e *Expedition) loadContextSection() string {
	ctx, err := ReadContextFiles(e.Continent)
	if err != nil {
		e.Logger.Warn("context injection failed: %v", err)
		return ""
	}
	return ctx
}

// Run executes the expedition with timeout and streaming output.
func (e *Expedition) Run(ctx context.Context) (string, error) {
	if e.DataOut == nil {
		e.DataOut = io.Discard
	}
	prompt := e.BuildPrompt()

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
				attribute.String("claude.model", model),
				attribute.Int("expedition.number", e.Number),
				attribute.Int("claude.timeout_sec", e.Config.TimeoutSec),
			}, platform.GenAISpanAttrs(model)...)...,
		),
	)
	defer invokeSpan.End()

	claudeCmd := e.Config.ClaudeCmd
	if claudeCmd == "" {
		claudeCmd = platform.DefaultClaudeCmd
	}

	newCmd := e.makeCmd
	if newCmd == nil {
		newCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return platform.NewShellCmd(ctx, name, args...)
		}
	}

	cmd := newCmd(expCtx, claudeCmd,
		"--model", model,
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
	go watchFlag(watchCtx, workDir, func(issue, title string) {
		e.setCurrentIssue(issue)
		invokeSpan.AddEvent("issue.picked",
			trace.WithAttributes(
				attribute.String("issue_id", issue),
				attribute.String("issue_title", title),
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
		watchInbox(watchCtx, e.Continent, func(dm domain.DMail) {
			if seenFiles[dm.Name] {
				return
			}
			seenFiles[dm.Name] = true
			domain.LogBanner(e.Logger, domain.BannerRecv, dm.Kind, dm.Name, dm.Description)
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
			invokeSpan.SetAttributes(attribute.StringSlice("stream.raw_events", rawEvents))
		}
		if result != nil && result.SessionID != "" {
			invokeSpan.SetAttributes(platform.GenAISessionAttrs(result.SessionID)...)
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
	if e.Config.OutputFormat != "json" {
		fmt.Fprintln(e.errWriter())
	}

	if expCtx.Err() == context.DeadlineExceeded {
		invokeSpan.AddEvent("expedition.timeout",
			trace.WithAttributes(attribute.String("timeout", timeout.String())),
		)
		return output.String(), fmt.Errorf("timeout after %v", timeout)
	}
	if ctx.Err() == context.Canceled {
		return output.String(), fmt.Errorf("interrupted")
	}

	if readError != nil {
		return output.String(), fmt.Errorf("stream read: %w", readError)
	}

	return output.String(), err
}

// ReadContextFiles reads all .md files from .expedition/context/ and
// concatenates them into a single string for prompt injection.
func ReadContextFiles(continent string) (string, error) {
	dir := domain.ContextDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
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
	for _, e := range sortedEntries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", fmt.Errorf("reading context file %s: %w", e.Name(), err)
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		buf.WriteString(fmt.Sprintf("### %s\n\n", name))
		buf.Write(content)
		buf.WriteString("\n\n")
	}
	return buf.String(), nil
}
