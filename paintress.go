package paintress

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const maxConsecutiveFailures = 3
const gradientMax = 5

var (
	errGommage  = errors.New("gommage: consecutive failures exceeded threshold")
	errComplete = errors.New("expedition complete: no remaining issues")
)

type Paintress struct {
	config    Config
	logDir    string
	Logger    *Logger
	devServer *DevServer
	gradient  *GradientGauge
	reserve   *ReserveParty
	pool      *WorktreePool // nil when --workers=0
	notifier  Notifier
	approver  Approver

	// Swarm Mode: atomic counters for concurrent worker access
	expCounter          atomic.Int64
	totalAttempted      atomic.Int64
	totalSuccess        atomic.Int64
	totalSkipped        atomic.Int64
	totalFailed         atomic.Int64
	totalBugs           atomic.Int64
	consecutiveFailures atomic.Int64

	// Swarm Mode: mutex-protected shared resources
	flagMu sync.Mutex
}

func NewPaintress(cfg Config, logger *Logger) *Paintress {
	if logger == nil {
		logger = NewLogger(nil, false)
	}
	logDir := filepath.Join(cfg.Continent, ".expedition", ".run", "logs")
	os.MkdirAll(logDir, 0755)

	// Reserve Party: parse model string for reserves
	// Format: "opus" or "opus,sonnet,haiku"
	models := strings.Split(cfg.Model, ",")
	primary := strings.TrimSpace(models[0])
	var reserves []string
	for _, m := range models[1:] {
		m = strings.TrimSpace(m)
		if m != "" {
			reserves = append(reserves, m)
		}
	}

	devDir := cfg.DevDir
	if devDir == "" {
		devDir = cfg.Continent
	}

	// Wire notifier based on config
	var notifier Notifier
	if cfg.NotifyCmd != "" {
		notifier = NewCmdNotifier(cfg.NotifyCmd)
	} else {
		notifier = &LocalNotifier{}
	}

	// Wire approver based on config
	var approver Approver
	switch {
	case cfg.AutoApprove:
		approver = &AutoApprover{}
	case cfg.ApproveCmd != "":
		approver = NewCmdApprover(cfg.ApproveCmd)
	default:
		approver = NewStdinApprover()
	}

	p := &Paintress{
		config:   cfg,
		logDir:   logDir,
		Logger:   logger,
		gradient: NewGradientGauge(gradientMax),
		reserve:  NewReserveParty(primary, reserves, logger),
		notifier: notifier,
		approver: approver,
	}

	if !cfg.NoDev {
		p.devServer = NewDevServer(
			cfg.DevCmd, cfg.DevURL, devDir,
			filepath.Join(logDir, "dev-server.log"),
			logger,
		)
	} else {
		p.config.DevURL = ""
	}

	return p
}

func (p *Paintress) Run(ctx context.Context) int {
	ctx, rootSpan := tracer.Start(ctx, "paintress.run",
		trace.WithAttributes(
			attribute.String("continent", p.config.Continent),
			attribute.Int("max_expeditions", p.config.MaxExpeditions),
			attribute.Int("workers", p.config.Workers),
			attribute.String("model", p.config.Model),
			attribute.Bool("dry_run", p.config.DryRun),
		),
	)
	defer rootSpan.End()

	logPath := filepath.Join(p.logDir, fmt.Sprintf("paintress-%s.log", time.Now().Format("20060102")))
	if err := p.Logger.SetLogFile(logPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: log file: %v\n", err)
	}
	defer p.Logger.CloseLogFile()

	monolith := ReadFlag(p.config.Continent)

	p.printBanner()
	p.Logger.Info("%s", fmt.Sprintf(Msg("continent"), p.config.Continent))
	p.Logger.Info("%s", fmt.Sprintf(Msg("monolith_reads"), monolith.Remaining))
	p.Logger.Info("%s", fmt.Sprintf(Msg("max_expeditions"), p.config.MaxExpeditions))
	p.Logger.Info("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))
	p.Logger.Info("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatForPrompt()))
	p.Logger.Info("%s", fmt.Sprintf(Msg("timeout_info"), p.config.TimeoutSec))
	claudeCmd := p.config.ClaudeCmd
	if claudeCmd == "" {
		claudeCmd = DefaultClaudeCmd
	}
	if claudeCmd != DefaultClaudeCmd {
		p.Logger.Info("%s", fmt.Sprintf(Msg("claude_cmd_info"), claudeCmd))
	}
	if p.config.DryRun {
		p.Logger.Warn("%s", Msg("dry_run"))
	}
	fmt.Fprintln(p.Logger.Writer())

	// Start dev server (stays alive across expeditions)
	if !p.config.DryRun && p.devServer != nil {
		if err := p.devServer.Start(ctx); err != nil {
			p.Logger.Warn("%s", fmt.Sprintf(Msg("devserver_warn"), err))
		}
		defer p.devServer.Stop()
	}

	// Initialize worktree pool if workers > 0
	if p.config.Workers > 0 {
		p.pool = NewWorktreePool(
			&localGitExecutor{},
			p.config.Continent,
			p.config.BaseBranch,
			p.config.SetupCmd,
			p.config.Workers,
		)
		if err := p.pool.Init(ctx); err != nil {
			p.Logger.Error("worktree pool init failed: %v", err)
			return 1
		}
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()
			p.pool.Shutdown(shutdownCtx)
		}()
	}

	// === Swarm Mode: reset run-scoped counters and launch workers ===
	p.totalAttempted.Store(0)
	p.totalSuccess.Store(0)
	p.totalSkipped.Store(0)
	p.totalFailed.Store(0)
	p.totalBugs.Store(0)
	p.consecutiveFailures.Store(0)

	startExp := monolith.LastExpedition + 1
	p.expCounter.Store(int64(startExp))

	// Pre-flight Lumina scan (once, before workers start)
	luminas := ScanJournalsForLumina(p.config.Continent)
	if len(luminas) > 0 {
		p.Logger.OK("%s", fmt.Sprintf(Msg("lumina_extracted"), len(luminas)))
	}

	// Pre-flight HIGH severity gate (once, before workers start).
	// This prevents concurrent StdinApprover reads when workers > 1.
	// Fail closed: if inbox cannot be read, abort rather than skip the gate.
	preflightInbox, scanErr := ScanInbox(p.config.Continent)
	if scanErr != nil {
		p.Logger.Error("inbox scan failed (fail-closed): %v", scanErr)
		return 1
	}
	if highDMails := FilterHighSeverity(preflightInbox); len(highDMails) > 0 {
		names := make([]string, len(highDMails))
		for i, dm := range highDMails {
			names[i] = dm.Name
		}
		msg := fmt.Sprintf("HIGH severity D-Mail detected: %s", strings.Join(names, ", "))
		p.Logger.Warn("%s", msg)

		if err := p.notifier.Notify(ctx, "Paintress", msg); err != nil {
			p.Logger.Warn("notification failed: %v", err)
		}

		approved, err := p.approver.RequestApproval(ctx, msg)
		if err != nil {
			p.Logger.Warn("approval request failed: %v", err)
		}
		if !approved {
			p.Logger.Warn("all expeditions aborted: HIGH severity D-Mail denied")
			return 0
		}
	}

	g, gCtx := errgroup.WithContext(ctx)
	workerCount := max(p.config.Workers, 1)

	for i := range workerCount {
		g.Go(func() error {
			return p.runWorker(gCtx, i, startExp, luminas)
		})
	}

	err := g.Wait()

	fmt.Fprintln(p.Logger.Writer())
	p.printSummary()

	switch {
	case errors.Is(err, errComplete):
		return 0
	case errors.Is(err, errGommage):
		return 1
	case ctx.Err() != nil:
		return 130
	case err != nil:
		return 1
	default:
		return 0
	}
}

// runWorker is the expedition loop that each worker goroutine runs.
// Each worker atomically claims expedition numbers and processes them
// until MaxExpeditions is reached, context is cancelled, or a sentinel
// error (errGommage/errComplete) terminates the group.
func (p *Paintress) runWorker(ctx context.Context, workerID int, startExp int, luminas []Lumina) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		exp := int(p.expCounter.Add(1) - 1)
		if exp >= startExp+p.config.MaxExpeditions {
			return nil
		}

		p.totalAttempted.Add(1)
		p.Logger.Exp("%s", fmt.Sprintf(Msg("departing"), exp))
		p.reserve.TryRecoverPrimary()
		p.Logger.Info("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatForPrompt()))
		p.Logger.Info("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))

		model := p.reserve.ActiveModel()
		expCtx, expSpan := tracer.Start(ctx, "expedition",
			trace.WithAttributes(
				attribute.Int("expedition.number", exp),
				attribute.Int("worker.id", workerID),
				attribute.String("model", model),
			),
		)

		var workDir string
		if p.pool != nil {
			_, acqSpan := tracer.Start(expCtx, "worktree.acquire")
			workDir = p.pool.Acquire()
			acqSpan.End()
		}
		releaseWorkDir := func() {
			if p.pool != nil && workDir != "" {
				_, relSpan := tracer.Start(expCtx, "worktree.release")
				rCtx, rCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer rCancel()
				if err := p.pool.Release(rCtx, workDir); err != nil {
					p.Logger.Warn("worktree release: %v", err)
				}
				workDir = ""
				relSpan.End()
			}
		}

		// Scan inbox for expedition prompt data (gate already ran in pre-flight)
		inboxDMails, scanErr := ScanInbox(p.config.Continent)
		if scanErr != nil {
			p.Logger.Warn("inbox scan for expedition #%d: %v", exp, scanErr)
		}

		expedition := &Expedition{
			Number:      exp,
			Continent:   p.config.Continent,
			WorkDir:     workDir,
			Config:      p.config,
			LogDir:      p.logDir,
			Logger:      p.Logger,
			Luminas:     luminas,
			Gradient:    p.gradient,
			Reserve:     p.reserve,
			InboxDMails: inboxDMails,
			Notifier:    p.notifier,
		}

		if p.config.DryRun {
			promptFile := filepath.Join(p.logDir, fmt.Sprintf("expedition-%03d-prompt.md", exp))
			if err := os.WriteFile(promptFile, []byte(expedition.BuildPrompt()), 0644); err != nil {
				p.Logger.Error("failed to write dry-run prompt: %v", err)
				releaseWorkDir()
				expSpan.End()
				continue
			}
			p.Logger.Warn("%s", fmt.Sprintf(Msg("dry_run_prompt"), promptFile))
			p.totalSuccess.Add(1)
			releaseWorkDir()
			expSpan.End()
			continue
		}

		p.Logger.Info("%s", fmt.Sprintf(Msg("sending"), p.reserve.ActiveModel()))
		expStart := time.Now()
		output, err := expedition.Run(expCtx)
		expElapsed := time.Since(expStart)

		if err != nil {
			if ctx.Err() != nil {
				releaseWorkDir()
				expSpan.End()
				return nil
			}
			p.Logger.Error("%s", fmt.Sprintf(Msg("exp_failed"), exp, err))
			if strings.Contains(err.Error(), "timeout") {
				prevModel := p.reserve.ActiveModel()
				p.reserve.ForceReserve()
				newModel := p.reserve.ActiveModel()
				if newModel != prevModel {
					expSpan.AddEvent("model.switched",
						trace.WithAttributes(
							attribute.String("from", prevModel),
							attribute.String("to", newModel),
						),
					)
				}
			}
			p.gradient.Discharge()
			p.flagMu.Lock()
			p.writeFlag(exp, "error", "failed", "?")
			p.flagMu.Unlock()
			WriteJournal(p.config.Continent, &ExpeditionReport{
				Expedition: exp, IssueID: "?", IssueTitle: "?",
				MissionType: "?", Status: "failed", Reason: err.Error(),
				FailureType: "blocker",
				PRUrl:       "none", BugIssues: "none",
			})
			p.consecutiveFailures.Add(1)
			p.totalFailed.Add(1)
		} else {
			_, parseSpan := tracer.Start(expCtx, "report.parse")
			report, status := ParseReport(output, exp)
			parseSpan.End()

			switch status {
			case StatusComplete:
				expSpan.AddEvent("expedition.complete",
					trace.WithAttributes(attribute.String("status", "all_complete")),
				)
				releaseWorkDir()
				expSpan.End()
				p.Logger.OK("%s", Msg("all_complete"))
				p.flagMu.Lock()
				p.writeFlag(exp, "all", "complete", "0")
				p.flagMu.Unlock()
				return errComplete
			case StatusParseError:
				p.Logger.Warn("%s", Msg("report_parse_fail"))
				p.Logger.Warn("%s", fmt.Sprintf(Msg("output_check"), p.logDir, exp))
				p.gradient.Decay()
				p.flagMu.Lock()
				p.writeFlag(exp, "?", "parse_error", "?")
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, &ExpeditionReport{
					Expedition: exp, IssueID: "?", IssueTitle: "?",
					MissionType: "?", Status: "parse_error", Reason: "report markers not found",
					FailureType: "blocker",
					PRUrl:       "none", BugIssues: "none",
				})
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)
			case StatusSuccess:
				expSpan.AddEvent("expedition.complete",
					trace.WithAttributes(
						attribute.String("status", "success"),
						attribute.String("issue_id", report.IssueID),
						attribute.String("mission_type", report.MissionType),
					),
				)
				p.handleSuccess(report)
				p.gradient.Charge()
				if report.PRUrl != "" && report.PRUrl != "none" && p.config.ReviewCmd != "" {
					totalTimeout := time.Duration(p.config.TimeoutSec) * time.Second
					remaining := totalTimeout - expElapsed
					if remaining > 0 {
						p.runReviewLoop(ctx, report, remaining, workDir)
					}
				}
				p.flagMu.Lock()
				p.writeFlag(exp, report.IssueID, "success", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				// D-Mail: send report and archive processed inbox d-mails (best-effort)
				if dm := NewReportDMail(report); dm.Name != "" {
					if err := SendDMail(p.config.Continent, dm); err != nil {
						p.Logger.Warn("dmail send: %v", err)
					}
				}
				for _, dm := range expedition.InboxDMails {
					if err := ArchiveInboxDMail(p.config.Continent, dm.Name); err != nil {
						p.Logger.Warn("dmail archive: %v", err)
					}
				}
				p.consecutiveFailures.Store(0)
				p.totalSuccess.Add(1)
			case StatusSkipped:
				p.Logger.Warn("%s", fmt.Sprintf(Msg("issue_skipped"), report.IssueID, report.Reason))
				p.gradient.Decay()
				p.flagMu.Lock()
				p.writeFlag(exp, report.IssueID, "skipped", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.totalSkipped.Add(1)
			case StatusFailed:
				p.Logger.Error("%s", fmt.Sprintf(Msg("issue_failed"), report.IssueID, report.Reason))
				p.gradient.Discharge()
				p.flagMu.Lock()
				p.writeFlag(exp, report.IssueID, "failed", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)
			}
		}

		if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) {
			expSpan.AddEvent("gommage",
				trace.WithAttributes(attribute.Int("consecutive_failures", maxConsecutiveFailures)),
			)
			releaseWorkDir()
			expSpan.End()
			p.Logger.Error("%s", fmt.Sprintf(Msg("gommage"), maxConsecutiveFailures))
			return errGommage
		}

		releaseWorkDir()
		expSpan.End()
		if p.pool == nil {
			gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
			gitCmd.Dir = p.config.Continent
			_ = gitCmd.Run()
		}

		p.Logger.Info("%s", Msg("cooldown"))
		select {
		case <-time.After(10 * time.Second):
		case <-ctx.Done():
			return nil
		}
	}
}

// runReviewLoop executes the code review command and, if comments are found,
// runs a lightweight Claude Code session to fix them. Repeats up to maxReviewCycles.
// Remaining review insights are appended to the report for journal recording.
//
// The timeout budget only counts time spent on Claude Code (reviewfix) phases.
// Review command execution (e.g. codex) does not consume the budget, so slow
// external review services do not eat into the fix time allowance.
func (p *Paintress) runReviewLoop(ctx context.Context, report *ExpeditionReport, budget time.Duration, workDir string) {
	ctx, loopSpan := tracer.Start(ctx, "review.loop",
		trace.WithAttributes(
			attribute.String("pr_url", report.PRUrl),
			attribute.String("branch", report.Branch),
		),
	)
	defer loopSpan.End()

	// Resolve execution directory: worktree path when pool is active, Continent otherwise.
	reviewDir := workDir
	if reviewDir == "" {
		reviewDir = p.config.Continent
	}

	// budget tracks only Claude Code (fix phase) execution time.
	// Review command time is excluded so slow external services
	// do not eat into the fix allowance.
	var consumed time.Duration

	// Each review command gets timeout / maxReviewCycles, with a floor
	// so that very short --timeout values don't instantly cancel reviews.
	reviewTimeout := max(
		time.Duration(p.config.TimeoutSec)*time.Second/time.Duration(maxReviewCycles),
		minReviewTimeout,
	)
	var lastComments string
	for cycle := 1; cycle <= maxReviewCycles; cycle++ {
		// Check parent context before starting each cycle
		if ctx.Err() != nil {
			if lastComments != "" {
				if report.Insight != "" {
					report.Insight += " | "
				}
				report.Insight += "Review interrupted: " + summarizeReview(lastComments)
			}
			p.Logger.Warn("%s", fmt.Sprintf(Msg("review_error"), ctx.Err()))
			return
		}

		p.Logger.Info("%s", fmt.Sprintf(Msg("review_running"), cycle, maxReviewCycles))

		// Review phase — bounded by reviewTimeout, does NOT consume budget
		_, revSpan := tracer.Start(ctx, "review.command",
			trace.WithAttributes(attribute.Int("cycle", cycle)),
		)
		reviewCtx, reviewCancel := context.WithTimeout(ctx, reviewTimeout)
		result, err := RunReview(reviewCtx, p.config.ReviewCmd, reviewDir)
		reviewCancel()
		if err != nil {
			revSpan.End()
			if lastComments != "" {
				if report.Insight != "" {
					report.Insight += " | "
				}
				report.Insight += "Review interrupted: " + summarizeReview(lastComments)
			}
			p.Logger.Warn("%s", fmt.Sprintf(Msg("review_error"), err))
			return
		}

		revSpan.SetAttributes(attribute.Bool("passed", result.Passed))
		revSpan.End()

		if result.Passed {
			p.Logger.OK("%s", Msg("review_passed"))
			return
		}

		lastComments = result.Comments
		p.Logger.Warn("%s", fmt.Sprintf(Msg("review_comments"), cycle))

		// Validate branch before attempting fix
		branch := strings.TrimSpace(report.Branch)
		if branch == "" || strings.EqualFold(branch, "none") {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Reviewfix skipped: no valid branch"
			return
		}

		// Ensure we are on the PR branch before applying fixes
		gitCtx, gitCancel := context.WithTimeout(ctx, gitCmdTimeout)
		gitCmd := exec.CommandContext(gitCtx, "git", "checkout", branch)
		gitCmd.Dir = reviewDir
		err = gitCmd.Run()
		gitCancel()
		if err != nil {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += fmt.Sprintf("Reviewfix skipped: checkout %s failed: %v", branch, err)
			return
		}

		// Check remaining budget before fix phase
		remaining := budget - consumed
		if remaining <= 0 {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Review not fully resolved: " + summarizeReview(lastComments)
			p.Logger.Warn("%s", Msg("review_limit"))
			return
		}

		// Fix phase — bounded by remaining budget (only this phase consumes it)
		fixCtx, fixCancel := context.WithTimeout(ctx, remaining)

		prompt := BuildReviewFixPrompt(branch, result.Comments)

		claudeCmd := p.config.ClaudeCmd
		if claudeCmd == "" {
			claudeCmd = DefaultClaudeCmd
		}

		model := p.reserve.ActiveModel()
		_, fixSpan := tracer.Start(fixCtx, "reviewfix.claude",
			trace.WithAttributes(
				attribute.Int("cycle", cycle),
				attribute.String("model", model),
			),
		)

		cmd := exec.CommandContext(fixCtx, claudeCmd,
			"--model", model,
			"--continue",
			"--dangerously-skip-permissions",
			"--print",
			"-p", prompt,
		)
		cmd.Dir = reviewDir
		cmd.WaitDelay = 3 * time.Second

		p.Logger.Info("%s", fmt.Sprintf(Msg("reviewfix_running"), model))
		start := time.Now()
		out, err := cmd.CombinedOutput()
		consumed += time.Since(start)
		fixSpan.End()
		fixCancel()

		if err != nil {
			p.Logger.Warn("%s", fmt.Sprintf(Msg("reviewfix_error"), err))
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Reviewfix failed: " + summarizeReview(string(out))
			return
		}
	}

	// All review-fix cycles exhausted — record remaining insights
	if report.Insight != "" {
		report.Insight += " | "
	}
	report.Insight += "Review not fully resolved: " + summarizeReview(lastComments)
	p.Logger.Warn("%s", Msg("review_limit"))
}

func (p *Paintress) handleSuccess(report *ExpeditionReport) {
	if report.MissionType == "verify" {
		p.Logger.QA("%s: %s", report.IssueID, report.IssueTitle)
		if report.BugsFound > 0 {
			p.Logger.QA("%s", fmt.Sprintf(Msg("qa_bugs"), report.BugsFound, report.BugIssues))
			p.totalBugs.Add(int64(report.BugsFound))
		} else {
			p.Logger.QA("%s", Msg("qa_all_pass"))
		}
	} else {
		p.Logger.OK("%s: %s [%s]", report.IssueID, report.IssueTitle, report.MissionType)
	}
	if report.PRUrl != "" && report.PRUrl != "none" {
		p.Logger.OK("PR: %s", report.PRUrl)
	}
	if report.Remaining != "" {
		p.Logger.Info("%s", fmt.Sprintf(Msg("monolith_reads"), report.Remaining))
	}
}

func (p *Paintress) printBanner() {
	w := p.Logger.Writer()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "╔══════════════════════════════════════════════╗")
	fmt.Fprintln(w, "║          The Paintress awakens               ║")
	fmt.Fprintln(w, "╚══════════════════════════════════════════════╝")
	fmt.Fprintln(w)
}

// writeFlag writes the flag checkpoint only if expNum is greater than the
// current checkpoint. This ensures monotonic progression when workers
// complete out of order. Caller must hold p.flagMu.
func (p *Paintress) writeFlag(expNum int, issueID, status, remaining string) {
	current := ReadFlag(p.config.Continent)
	if expNum <= current.LastExpedition {
		return
	}
	WriteFlag(p.config.Continent, expNum, issueID, status, remaining)
}

// RunSummary holds the results of a paintress loop run.
type RunSummary struct {
	Total    int64  `json:"total"`
	Success  int64  `json:"success"`
	Skipped  int64  `json:"skipped"`
	Failed   int64  `json:"failed"`
	Bugs     int64  `json:"bugs"`
	Gradient string `json:"gradient"`
}

// FormatSummaryJSON returns the summary as a JSON string.
func FormatSummaryJSON(s RunSummary) (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p *Paintress) printSummary() {
	total := p.totalAttempted.Load()

	if p.config.OutputFormat == "json" {
		summary := RunSummary{
			Total:    total,
			Success:  p.totalSuccess.Load(),
			Skipped:  p.totalSkipped.Load(),
			Failed:   p.totalFailed.Load(),
			Bugs:     p.totalBugs.Load(),
			Gradient: p.gradient.FormatLog(),
		}
		out, err := FormatSummaryJSON(summary)
		if err != nil {
			p.Logger.Error("json marshal: %v", err)
			return
		}
		fmt.Println(out)
		return
	}

	w := p.Logger.Writer()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "╔══════════════════════════════════════════════╗")
	fmt.Fprintln(w, "║          The Paintress rests                 ║")
	fmt.Fprintln(w, "╚══════════════════════════════════════════════╝")
	fmt.Fprintln(w)
	p.Logger.Info("%s", fmt.Sprintf(Msg("expeditions_sent"), total))
	p.Logger.OK("%s", fmt.Sprintf(Msg("success_count"), p.totalSuccess.Load()))
	p.Logger.Warn("%s", fmt.Sprintf(Msg("skipped_count"), p.totalSkipped.Load()))
	p.Logger.Error("%s", fmt.Sprintf(Msg("failed_count"), p.totalFailed.Load()))
	if p.totalBugs.Load() > 0 {
		p.Logger.QA("%s", fmt.Sprintf(Msg("bugs_count"), p.totalBugs.Load()))
	}
	fmt.Fprintln(w)
	p.Logger.Info("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatLog()))
	p.Logger.Info("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))
	fmt.Fprintln(w)
	p.Logger.Info("Flag:     %s", FlagPath(p.config.Continent))
	p.Logger.Info("Journals: %s", JournalDir(p.config.Continent))
	p.Logger.Info("Logs:     %s", p.logDir)
}
