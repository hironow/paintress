package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
	devServer *DevServer
	gradient  *GradientGauge
	reserve   *ReserveParty
	pool      *WorktreePool // nil when --workers=0

	// Swarm Mode: atomic counters for concurrent worker access
	expCounter          atomic.Int64
	totalSuccess        atomic.Int64
	totalSkipped        atomic.Int64
	totalFailed         atomic.Int64
	totalBugs           atomic.Int64
	consecutiveFailures atomic.Int64

	// Swarm Mode: mutex-protected shared resources
	flagMu sync.Mutex
}

func NewPaintress(cfg Config) *Paintress {
	logDir := filepath.Join(cfg.Continent, ".expedition", ".logs")
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

	return &Paintress{
		config:   cfg,
		logDir:   logDir,
		gradient: NewGradientGauge(gradientMax),
		reserve:  NewReserveParty(primary, reserves),
		devServer: NewDevServer(
			cfg.DevCmd, cfg.DevURL, devDir,
			filepath.Join(logDir, "dev-server.log"),
		),
	}
}

func (p *Paintress) Run(ctx context.Context) int {
	logPath := filepath.Join(p.logDir, fmt.Sprintf("paintress-%s.log", time.Now().Format("20060102")))
	if err := InitLogFile(logPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: log file: %v\n", err)
	}
	defer CloseLogFile()

	// Write mission.md in the active language before expeditions start
	if err := WriteMission(p.config.Continent); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: mission file: %v\n", err)
	}

	monolith := ReadFlag(p.config.Continent)

	p.printBanner()
	LogInfo("%s", fmt.Sprintf(Msg("continent"), p.config.Continent))
	LogInfo("%s", fmt.Sprintf(Msg("monolith_reads"), monolith.Remaining))
	LogInfo("%s", fmt.Sprintf(Msg("max_expeditions"), p.config.MaxExpeditions))
	LogInfo("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))
	LogInfo("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatForPrompt()))
	LogInfo("%s", fmt.Sprintf(Msg("timeout_info"), p.config.TimeoutSec))
	claudeCmd := p.config.ClaudeCmd
	if claudeCmd == "" {
		claudeCmd = defaultClaudeCmd
	}
	if claudeCmd != defaultClaudeCmd {
		LogInfo("%s", fmt.Sprintf(Msg("claude_cmd_info"), claudeCmd))
	}
	if p.config.DryRun {
		LogWarn("%s", Msg("dry_run"))
	}
	fmt.Println()

	// Start dev server (stays alive across expeditions)
	if !p.config.DryRun {
		if err := p.devServer.Start(ctx); err != nil {
			LogWarn("%s", fmt.Sprintf(Msg("devserver_warn"), err))
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
			LogError("worktree pool init failed: %v", err)
			return 1
		}
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer shutdownCancel()
			p.pool.Shutdown(shutdownCtx)
		}()
	}

	// === Swarm Mode: launch workers ===
	startExp := monolith.LastExpedition + 1
	p.expCounter.Store(int64(startExp))

	// Pre-flight Lumina scan (once, before workers start)
	luminas := ScanJournalsForLumina(p.config.Continent)
	if len(luminas) > 0 {
		LogOK("%s", fmt.Sprintf(Msg("lumina_extracted"), len(luminas)))
		WriteLumina(p.config.Continent, luminas)
	}

	g, gCtx := errgroup.WithContext(ctx)
	workerCount := max(p.config.Workers, 1)

	for i := range workerCount {
		g.Go(func() error {
			return p.runWorker(gCtx, i, startExp, luminas)
		})
	}

	err := g.Wait()

	fmt.Println()
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

		LogExp("%s", fmt.Sprintf(Msg("departing"), exp))
		p.reserve.TryRecoverPrimary()
		LogInfo("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatForPrompt()))
		LogInfo("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))

		var workDir string
		if p.pool != nil {
			workDir = p.pool.Acquire()
		}
		releaseWorkDir := func() {
			if p.pool != nil && workDir != "" {
				rCtx, rCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer rCancel()
				if err := p.pool.Release(rCtx, workDir); err != nil {
					LogWarn("worktree release: %v", err)
				}
				workDir = ""
			}
		}

		expedition := &Expedition{
			Number:    exp,
			Continent: p.config.Continent,
			WorkDir:   workDir,
			Config:    p.config,
			LogDir:    p.logDir,
			Luminas:   luminas,
			Gradient:  p.gradient,
			Reserve:   p.reserve,
		}

		if p.config.DryRun {
			promptFile := filepath.Join(p.logDir, fmt.Sprintf("expedition-%03d-prompt.md", exp))
			os.WriteFile(promptFile, []byte(expedition.BuildPrompt()), 0644)
			LogWarn("%s", fmt.Sprintf(Msg("dry_run_prompt"), promptFile))
			p.totalSuccess.Add(1)
			releaseWorkDir()
			return nil
		}

		LogInfo("%s", fmt.Sprintf(Msg("sending"), p.reserve.ActiveModel()))
		expStart := time.Now()
		output, err := expedition.Run(ctx)
		expElapsed := time.Since(expStart)

		if err != nil {
			if ctx.Err() == context.Canceled {
				releaseWorkDir()
				return nil
			}
			LogError("%s", fmt.Sprintf(Msg("exp_failed"), exp, err))
			if strings.Contains(err.Error(), "timeout") {
				p.reserve.ForceReserve()
			}
			p.gradient.Discharge()
			p.flagMu.Lock()
			WriteFlag(p.config.Continent, exp, "error", "failed", "?")
			p.flagMu.Unlock()
			WriteJournal(p.config.Continent, &ExpeditionReport{
				Expedition: exp, IssueID: "?", IssueTitle: "?",
				MissionType: "?", Status: "failed", Reason: err.Error(),
				PRUrl: "none", BugIssues: "none",
			})
			p.consecutiveFailures.Add(1)
			p.totalFailed.Add(1)
		} else {
			report, status := ParseReport(output, exp)
			switch status {
			case StatusComplete:
				releaseWorkDir()
				LogOK("%s", Msg("all_complete"))
				p.flagMu.Lock()
				WriteFlag(p.config.Continent, exp, "all", "complete", "0")
				p.flagMu.Unlock()
				p.totalSkipped.Add(1)
				return errComplete
			case StatusParseError:
				LogWarn("%s", Msg("report_parse_fail"))
				LogWarn("%s", fmt.Sprintf(Msg("output_check"), p.logDir, exp))
				p.gradient.Decay()
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)
			case StatusSuccess:
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
				WriteFlag(p.config.Continent, exp, report.IssueID, "success", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.consecutiveFailures.Store(0)
				p.totalSuccess.Add(1)
			case StatusSkipped:
				LogWarn("%s", fmt.Sprintf(Msg("issue_skipped"), report.IssueID, report.Reason))
				p.gradient.Decay()
				p.flagMu.Lock()
				WriteFlag(p.config.Continent, exp, report.IssueID, "skipped", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.totalSkipped.Add(1)
			case StatusFailed:
				LogError("%s", fmt.Sprintf(Msg("issue_failed"), report.IssueID, report.Reason))
				p.gradient.Discharge()
				p.flagMu.Lock()
				WriteFlag(p.config.Continent, exp, report.IssueID, "failed", report.Remaining)
				p.flagMu.Unlock()
				WriteJournal(p.config.Continent, report)
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)
			}
		}

		if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) {
			releaseWorkDir()
			LogError("%s", fmt.Sprintf(Msg("gommage"), maxConsecutiveFailures))
			return errGommage
		}

		releaseWorkDir()
		if p.pool == nil {
			gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
			gitCmd.Dir = p.config.Continent
			_ = gitCmd.Run()
		}

		LogInfo("%s", Msg("cooldown"))
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
			LogWarn("%s", fmt.Sprintf(Msg("review_error"), ctx.Err()))
			return
		}

		LogInfo("%s", fmt.Sprintf(Msg("review_running"), cycle, maxReviewCycles))

		// Review phase — bounded by reviewTimeout, does NOT consume budget
		reviewCtx, reviewCancel := context.WithTimeout(ctx, reviewTimeout)
		result, err := RunReview(reviewCtx, p.config.ReviewCmd, reviewDir)
		reviewCancel()
		if err != nil {
			if lastComments != "" {
				if report.Insight != "" {
					report.Insight += " | "
				}
				report.Insight += "Review interrupted: " + summarizeReview(lastComments)
			}
			LogWarn("%s", fmt.Sprintf(Msg("review_error"), err))
			return
		}

		if result.Passed {
			LogOK("%s", Msg("review_passed"))
			return
		}

		lastComments = result.Comments
		LogWarn("%s", fmt.Sprintf(Msg("review_comments"), cycle))

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
			LogWarn("%s", Msg("review_limit"))
			return
		}

		// Fix phase — bounded by remaining budget (only this phase consumes it)
		fixCtx, fixCancel := context.WithTimeout(ctx, remaining)

		prompt := BuildReviewFixPrompt(branch, result.Comments)

		claudeCmd := p.config.ClaudeCmd
		if claudeCmd == "" {
			claudeCmd = defaultClaudeCmd
		}

		model := p.reserve.ActiveModel()
		cmd := exec.CommandContext(fixCtx, claudeCmd,
			"--model", model,
			"--continue",
			"--dangerously-skip-permissions",
			"--print",
			"-p", prompt,
		)
		cmd.Dir = reviewDir
		cmd.WaitDelay = 3 * time.Second

		LogInfo("%s", fmt.Sprintf(Msg("reviewfix_running"), model))
		start := time.Now()
		out, err := cmd.CombinedOutput()
		consumed += time.Since(start)
		fixCancel()

		if err != nil {
			LogWarn("%s", fmt.Sprintf(Msg("reviewfix_error"), err))
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
	LogWarn("%s", Msg("review_limit"))
}

func (p *Paintress) handleSuccess(report *ExpeditionReport) {
	if report.MissionType == "verify" {
		LogQA("%s: %s", report.IssueID, report.IssueTitle)
		if report.BugsFound > 0 {
			LogQA("%s", fmt.Sprintf(Msg("qa_bugs"), report.BugsFound, report.BugIssues))
			p.totalBugs.Add(int64(report.BugsFound))
		} else {
			LogQA("%s", Msg("qa_all_pass"))
		}
	} else {
		LogOK("%s: %s [%s]", report.IssueID, report.IssueTitle, report.MissionType)
	}
	if report.PRUrl != "" && report.PRUrl != "none" {
		LogOK("PR: %s", report.PRUrl)
	}
	if report.Remaining != "" {
		LogInfo("%s", fmt.Sprintf(Msg("monolith_reads"), report.Remaining))
	}
}

func (p *Paintress) printBanner() {
	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║          The Paintress awakens               ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Println()
}

func (p *Paintress) printSummary() {
	total := p.totalSuccess.Load() + p.totalFailed.Load() + p.totalSkipped.Load()
	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║          The Paintress rests                 ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Println()
	LogInfo("%s", fmt.Sprintf(Msg("expeditions_sent"), total))
	LogOK("%s", fmt.Sprintf(Msg("success_count"), p.totalSuccess.Load()))
	LogWarn("%s", fmt.Sprintf(Msg("skipped_count"), p.totalSkipped.Load()))
	LogError("%s", fmt.Sprintf(Msg("failed_count"), p.totalFailed.Load()))
	if p.totalBugs.Load() > 0 {
		LogQA("%s", fmt.Sprintf(Msg("bugs_count"), p.totalBugs.Load()))
	}
	fmt.Println()
	LogInfo("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatLog()))
	LogInfo("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))
	fmt.Println()
	LogInfo("Flag:     %s", FlagPath(p.config.Continent))
	LogInfo("Lumina:   %s", LuminaPath(p.config.Continent))
	LogInfo("Journals: %s", JournalDir(p.config.Continent))
	LogInfo("Logs:     %s", p.logDir)
}
