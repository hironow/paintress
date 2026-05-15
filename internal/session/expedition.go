package session

import (
	"context"
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
	"github.com/hironow/paintress/internal/usecase/port"
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

	// capturedStderr is retained for API compatibility. The legacy Run()
	// captured `claude -p` stderr here; the deprecated stub leaves it
	// empty.
	capturedStderr string

	// makeCmd overrides command creation for testing. Retained for API
	// compatibility with the legacy Run(); the deprecated stub never
	// invokes it.
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
//
//nolint:unused // expires: 2026-07-15 — retained for the sub-B MCP-pivot commit which deletes the legacy MidMatched routing helpers after the ADR lands.
func (e *Expedition) appendMidHighName(name string) {
	e.midHighMu.Lock()
	defer e.midHighMu.Unlock()
	e.midHighNames = append(e.midHighNames, name)
}

// appendMidMatchedMail appends an issue-matched D-Mail (thread-safe).
//
//nolint:unused // expires: 2026-07-15 — retained for the sub-B MCP-pivot commit which deletes the legacy MidMatched routing helpers after the ADR lands.
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
// Retained for future reuse from the paintress MCP server tools; no
// longer invoked from the Go CLI control plane.
func (e *Expedition) BuildPrompt(ctxArgs ...context.Context) string {
	// Accept optional ctx for propagation; default to context.Background() for
	// backward compatibility with callers that don't pass a context (tests, dry-run).
	var ctx context.Context
	if len(ctxArgs) > 0 && ctxArgs[0] != nil {
		ctx = ctxArgs[0]
	} else {
		ctx = context.Background() // nosemgrep: gap050-session-context-background [permanent] — variadic backward compat: tests and dry-run callers omit ctx
	}
	projCfg, err := LoadProjectConfig(e.Continent)
	if err != nil {
		e.Logger.Warn("project config load failed: %v", err)
		projCfg = &domain.ProjectConfig{}
	}

	// loadInboxSection populates e.InboxDMails (via inboxOnce) and returns
	// the formatted inbox text. We pre-render the inbox first so we can
	// then derive the v1.1 HasEventSourcedContract flag from the same scan.
	inboxSection := e.loadInboxSection(ctx)

	data := domain.PromptData{
		Number:                  e.Number,
		Timestamp:               time.Now().Format("2006-01-02 15:04:05"),
		Bt:                      "`",
		Cb:                      "```",
		LuminaSection:           harness.FormatLuminaForPrompt(e.Luminas),
		GradientSection:         e.Gradient.FormatForPrompt(),
		ReserveSection:          e.Reserve.FormatForPrompt(),
		BaseBranch:              e.Config.BaseBranch,
		DevURL:                  e.Config.DevURL,
		ContextSection:          e.loadContextSection() + e.resumeSection(),
		InboxSection:            inboxSection,
		LinearTeam:              projCfg.TrackerTeam(),
		LinearProject:           projCfg.TrackerProject(),
		MissionSection:          harness.MissionText(harness.MustDefaultPromptRegistry(), domain.Lang, e.Target != nil),
		WaveTarget:              e.Target,
		HasEventSourcedContract: harness.HasEventSourcedContract(e.InboxDMails),
	}

	return harness.RenderExpeditionPrompt(harness.MustDefaultPromptRegistry(), domain.Lang, data)
}

func (e *Expedition) loadInboxSection(ctx context.Context) string {
	e.inboxOnce.Do(func() {
		if len(e.InboxDMails) > 0 {
			return // already loaded externally (e.g., by HIGH severity gate)
		}
		dmails, err := ScanInbox(ctx, e.Continent)
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

// ErrMCPPivotDeprecated is returned by Expedition.Run() now that the
// LLM invocation layer has moved to a human-initiated claude code
// interactive session per refs/issues/0027 Phase 1 (jun15 MCP pivot).
//
// Callers (paintress run / sweep / etc.) should surface this error
// and direct the operator to launch claude code with:
//
//	claude --plugin-dir ./plugins/paintress \
//	       --mcp-config '{"paintress":{"command":"paintress","args":["mcp"]}}'
//
// then invoke the /expedition-next slash command.
var ErrMCPPivotDeprecated = errors.New(
	"paintress Go CLI expedition runner deprecated post jun15 MCP pivot: " +
		"use claude code /expedition-next skill (refs/issues/0027)",
)

// Run returns ErrMCPPivotDeprecated. The previous implementation
// invoked `claude -p` via exec.Command, which is forbidden post the
// jun15 MCP pivot (refs/issues/0027 §5 billing boundary). LLM
// inference now happens inside a human-initiated claude code session
// driven by the paintress MCP server (`paintress mcp` subcommand)
// plus the /expedition-next slash command defined in
// plugins/paintress/skills/expedition-next/SKILL.md.
//
// The original Run() body (~335 lines) built a prompt, invoked
// `claude -p` via exec.Command, parsed stream-json from stdout,
// watched flag.md + inbox/ for mid-expedition events, and terminated
// the OpenTelemetry span. The entire block was removed in the Phase 1
// completion commit; the helpers above (BuildPrompt /
// loadInboxSection / loadContextSection / resumeSection) are kept
// because tests and a follow-up MCP wiring commit still reference
// them, but they no longer execute `claude -p`.
func (e *Expedition) Run(_ context.Context) (string, error) {
	if e.Logger != nil {
		e.Logger.Warn("paintress: expedition.Run() is deprecated (refs/issues/0027 jun15 MCP pivot); use the claude code /expedition-next skill instead.")
	}
	return "", ErrMCPPivotDeprecated
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
		fmt.Fprintf(&buf, "### %s\n\n", name)
		buf.Write(content)
		buf.WriteString("\n\n")
		totalBytes += info.Size()
	}
	return buf.String(), nil
}
