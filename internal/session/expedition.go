package session

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hironow/paintress"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Expedition represents a single expedition into the Continent.
type Expedition struct {
	Number    int
	Continent string
	WorkDir   string // execution directory (worktree path or Continent)
	Config    paintress.Config
	LogDir    string
	Logger    *paintress.Logger
	DataOut   io.Writer          // stdout-equivalent for streaming Claude output
	Notifier  paintress.Notifier // for mid-expedition HIGH severity notifications

	// Game mechanics
	Luminas     []paintress.Lumina
	Gradient    *paintress.GradientGauge
	Reserve     *paintress.ReserveParty
	InboxDMails []paintress.DMail // d-mails from inbox (for archiving after expedition)
	inboxOnce   sync.Once

	// Mid-expedition HIGH severity D-Mail tracking
	midHighMu    sync.Mutex
	midHighNames []string

	// Mid-expedition issue-matched D-Mail routing (MY-361)
	currentIssueMu  sync.Mutex
	currentIssue    string
	midMatchedMu    sync.Mutex
	midMatchedMails []paintress.DMail

	// makeCmd overrides command creation for testing. If nil, exec.CommandContext is used.
	makeCmd func(ctx context.Context, name string, args ...string) *exec.Cmd
}

// containsIssue reports whether issues contains target (case-insensitive).
func containsIssue(issues []string, target string) bool {
	if target == "" {
		return false
	}
	for _, id := range issues {
		if strings.EqualFold(id, target) {
			return true
		}
	}
	return false
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
func (e *Expedition) MidMatchedDMails() []paintress.DMail {
	e.midMatchedMu.Lock()
	defer e.midMatchedMu.Unlock()
	if len(e.midMatchedMails) == 0 {
		return []paintress.DMail{}
	}
	return append([]paintress.DMail(nil), e.midMatchedMails...)
}

// appendMidHighName appends a HIGH severity D-Mail name (thread-safe).
func (e *Expedition) appendMidHighName(name string) {
	e.midHighMu.Lock()
	defer e.midHighMu.Unlock()
	e.midHighNames = append(e.midHighNames, name)
}

// appendMidMatchedMail appends an issue-matched D-Mail (thread-safe).
func (e *Expedition) appendMidMatchedMail(dm paintress.DMail) {
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
		projCfg = &paintress.ProjectConfig{}
	}

	data := paintress.PromptData{
		Number:          e.Number,
		Timestamp:       time.Now().Format("2006-01-02 15:04:05"),
		Bt:              "`",
		Cb:              "```",
		LuminaSection:   paintress.FormatLuminaForPrompt(e.Luminas),
		GradientSection: e.Gradient.FormatForPrompt(),
		ReserveSection:  e.Reserve.FormatForPrompt(),
		BaseBranch:      e.Config.BaseBranch,
		DevURL:          e.Config.DevURL,
		ContextSection:  e.loadContextSection(),
		InboxSection:    e.loadInboxSection(),
		LinearTeam:      projCfg.Linear.Team,
		LinearProject:   projCfg.Linear.Project,
		MissionSection:  paintress.MissionText(),
	}

	tmplName := "expedition_en.md.tmpl"
	switch paintress.Lang {
	case "ja":
		tmplName = "expedition_ja.md.tmpl"
	case "fr":
		tmplName = "expedition_fr.md.tmpl"
	}

	var buf strings.Builder
	if err := paintress.ExpeditionTemplates.ExecuteTemplate(&buf, tmplName, data); err != nil {
		panic(fmt.Sprintf("prompt template execution failed: %v", err))
	}
	return buf.String()
}

func (e *Expedition) loadInboxSection() string {
	e.inboxOnce.Do(func() {
		if len(e.InboxDMails) > 0 {
			return // already loaded externally (e.g., by HIGH severity gate)
		}
		dmails, err := ScanInbox(e.Continent)
		if err != nil {
			e.Logger.Warn("inbox scan failed: %v", err)
			return
		}
		e.InboxDMails = dmails
	})
	return paintress.FormatDMailForPrompt(e.InboxDMails)
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
		e.DataOut = os.Stdout
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

	expCtx, invokeSpan := paintress.Tracer.Start(expCtx, "claude.invoke",
		trace.WithAttributes(
			attribute.String("model", model),
			attribute.Int("expedition.number", e.Number),
			attribute.Int("timeout_sec", e.Config.TimeoutSec),
			attribute.String("gen_ai.operation.name", "chat"),
			attribute.String("gen_ai.system", "anthropic"),
			attribute.String("gen_ai.request.model", model),
		),
	)
	defer invokeSpan.End()

	newCmd := e.makeCmd
	if newCmd == nil {
		newCmd = exec.CommandContext
	}

	claudeCmd := e.Config.ClaudeCmd
	if claudeCmd == "" {
		claudeCmd = paintress.DefaultClaudeCmd
	}

	cmd := newCmd(expCtx, claudeCmd,
		"--model", model,
		"--dangerously-skip-permissions",
		"--print",
		"-p", prompt,
	)
	workDir := e.WorkDir
	if workDir == "" {
		workDir = e.Continent
	}
	cmd.Dir = workDir

	if err := os.MkdirAll(filepath.Join(workDir, ".expedition", ".run"), 0755); err != nil {
		return "", fmt.Errorf("create expedition run dir: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe failed: %w", err)
	}
	cmd.Stderr = cmd.Stdout

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
		watchInbox(watchCtx, e.Continent, func(dm paintress.DMail) {
			if seenFiles[dm.Name] {
				return
			}
			seenFiles[dm.Name] = true
			if dm.Severity == "high" {
				e.appendMidHighName(dm.Name)
				e.Logger.Warn("HIGH severity d-mail received mid-expedition: %s", dm.Name)
				if e.Notifier != nil {
					_ = e.Notifier.Notify(watchCtx, "Paintress", fmt.Sprintf("HIGH severity D-Mail mid-expedition: %s", dm.Name))
				}
			} else {
				e.Logger.Info("Expedition #%d: d-mail received — %s (%s)", e.Number, dm.Name, dm.Kind)
			}
			if cur := e.getCurrentIssue(); cur != "" && containsIssue(dm.Issues, cur) {
				e.appendMidMatchedMail(dm)
				e.Logger.Info("Expedition #%d: d-mail routed to current issue %s — %s", e.Number, cur, dm.Name)
			}
		}, nil)
	}()

	// Streaming goroutine: tee to terminal + file + buffer + rate limit detection
	var output strings.Builder
	done := make(chan struct{})

	go func() {
		defer close(done)
		reader := bufio.NewReader(stdout)
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

		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				writer.Write(chunk)
				output.Write(chunk)

				if e.Reserve.CheckOutput(string(chunk)) {
					invokeSpan.AddEvent("rate_limit.detected")
				}
			}
			if err != nil {
				break
			}
		}
	}()

	<-done

	watchCancel()
	<-inboxDone

	err = cmd.Wait()
	if e.Config.OutputFormat != "json" {
		fmt.Fprintln(e.Logger.Writer())
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

	return output.String(), err
}

// ReadContextFiles reads all .md files from .expedition/context/ and
// concatenates them into a single string for prompt injection.
func ReadContextFiles(continent string) (string, error) {
	dir := paintress.ContextDir(continent)
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
