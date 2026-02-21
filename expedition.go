package paintress

import (
	"bufio"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

//go:embed templates/expedition_*.md.tmpl
var expeditionFS embed.FS

var expeditionTemplates = template.Must(
	template.ParseFS(expeditionFS, "templates/expedition_*.md.tmpl"),
)

// PromptData holds all dynamic values injected into the expedition prompt template.
type PromptData struct {
	Number          int
	Timestamp       string
	Bt              string // "`"
	Cb              string // "```"
	LuminaSection   string
	GradientSection string
	ReserveSection  string
	BaseBranch      string
	DevURL          string
	ContextSection  string
	InboxSection    string
	LinearTeam      string
	LinearProject   string
	MissionSection  string
}

// Expedition represents a single expedition into the Continent.
type Expedition struct {
	Number    int
	Continent string
	WorkDir   string // execution directory (worktree path or Continent)
	Config    Config
	LogDir    string
	Logger    *Logger
	DataOut   io.Writer // stdout-equivalent for streaming Claude output
	Notifier  Notifier  // for mid-expedition HIGH severity notifications

	// Game mechanics
	Luminas     []Lumina
	Gradient    *GradientGauge
	Reserve     *ReserveParty
	InboxDMails []DMail // d-mails from inbox (for archiving after expedition)
	inboxOnce   sync.Once

	// Mid-expedition HIGH severity D-Mail tracking
	midHighMu    sync.Mutex
	midHighNames []string

	// makeCmd overrides command creation for testing. If nil, exec.CommandContext is used.
	makeCmd func(ctx context.Context, name string, args ...string) *exec.Cmd
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
		projCfg = &ProjectConfig{}
	}

	data := PromptData{
		Number:          e.Number,
		Timestamp:       time.Now().Format("2006-01-02 15:04:05"),
		Bt:              "`",
		Cb:              "```",
		LuminaSection:   FormatLuminaForPrompt(e.Luminas),
		GradientSection: e.Gradient.FormatForPrompt(),
		ReserveSection:  e.Reserve.FormatForPrompt(),
		BaseBranch:      e.Config.BaseBranch,
		DevURL:          e.Config.DevURL,
		ContextSection:  e.loadContextSection(),
		InboxSection:    e.loadInboxSection(),
		LinearTeam:      projCfg.Linear.Team,
		LinearProject:   projCfg.Linear.Project,
		MissionSection:  MissionText(),
	}

	tmplName := "expedition_en.md.tmpl"
	switch Lang {
	case "ja":
		tmplName = "expedition_ja.md.tmpl"
	case "fr":
		tmplName = "expedition_fr.md.tmpl"
	}

	var buf strings.Builder
	if err := expeditionTemplates.ExecuteTemplate(&buf, tmplName, data); err != nil {
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
	return FormatDMailForPrompt(e.InboxDMails)
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
// The output streaming goroutine also feeds chunks to ReserveParty
// for rate-limit detection.
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

	// Use the active model from Reserve Party
	model := e.Reserve.ActiveModel()

	expCtx, invokeSpan := tracer.Start(expCtx, "claude.invoke",
		trace.WithAttributes(
			attribute.String("model", model),
			attribute.Int("expedition.number", e.Number),
			attribute.Int("timeout_sec", e.Config.TimeoutSec),
		),
	)
	defer invokeSpan.End()

	newCmd := e.makeCmd
	if newCmd == nil {
		newCmd = exec.CommandContext
	}

	claudeCmd := e.Config.ClaudeCmd
	if claudeCmd == "" {
		claudeCmd = DefaultClaudeCmd
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

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("%s start failed: %w", claudeCmd, err)
	}

	// Start flag.md watcher to detect issue selection in real-time
	watchCtx, watchCancel := context.WithCancel(expCtx)
	defer watchCancel()
	go watchFlag(watchCtx, e.Continent, func(issue, title string) {
		invokeSpan.AddEvent("issue.picked",
			trace.WithAttributes(
				attribute.String("issue_id", issue),
				attribute.String("issue_title", title),
			),
		)
		e.Logger.Info("Expedition #%d: issue picked — %s (%s)", e.Number, issue, title)
	}, nil)

	// Start inbox watcher to log d-mails arriving mid-expedition.
	// Mid-expedition arrivals are NOT appended to InboxDMails — they stay
	// in inbox/ and will be picked up by the next expedition's ScanInbox.
	// Only d-mails included in the prompt (initial scan) are archived.
	seenFiles := make(map[string]bool)
	for _, dm := range e.InboxDMails {
		seenFiles[dm.Name] = true
	}
	inboxDone := make(chan struct{})
	go func() {
		defer close(inboxDone)
		watchInbox(watchCtx, e.Continent, func(dm DMail) {
			if seenFiles[dm.Name] {
				return
			}
			seenFiles[dm.Name] = true
			if dm.Severity == "high" {
				e.midHighMu.Lock()
				e.midHighNames = append(e.midHighNames, dm.Name)
				e.midHighMu.Unlock()
				e.Logger.Warn("HIGH severity d-mail received mid-expedition: %s", dm.Name)
				if e.Notifier != nil {
					_ = e.Notifier.Notify(watchCtx, "Paintress", fmt.Sprintf("HIGH severity D-Mail mid-expedition: %s", dm.Name))
				}
			} else {
				e.Logger.Info("Expedition #%d: d-mail received — %s (%s)", e.Number, dm.Name, dm.Kind)
			}
		}, nil)
	}()

	// Streaming goroutine: tee to terminal + file + buffer + rate limit detection
	var output strings.Builder
	done := make(chan struct{})

	go func() {
		defer close(done)
		reader := bufio.NewReader(stdout)
		// In JSON output mode, stream to Logger (stderr) so DataOut (stdout)
		// stays machine-readable.
		// NOTE: This assumes Logger writes to a stream distinct from DataOut,
		// which is guaranteed by the CLI layer (NewLogger uses cmd.ErrOrStderr,
		// DataOut uses cmd.OutOrStdout). If a caller configures both to the
		// same writer, streaming output will interleave with JSON summary.
		streamDest := e.DataOut
		if e.Config.OutputFormat == "json" {
			streamDest = e.Logger.Writer()
		}
		writer := io.MultiWriter(streamDest, outFile)

		buf := make([]byte, 4096)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				writer.Write(chunk)
				output.Write(chunk)

				// Reserve Party: scan for rate limit signals in real-time
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

	// Stop watchers and join inbox watcher to ensure InboxDMails is
	// stable before callers iterate the slice for archiving.
	watchCancel()
	<-inboxDone

	err = cmd.Wait()
	fmt.Fprintln(e.Logger.Writer())

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
