package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const maxConsecutiveFailures = 3
const gradientMax = 5

type Paintress struct {
	config    Config
	logDir    string
	devServer *DevServer
	gradient  *GradientGauge
	reserve   *ReserveParty

	totalSuccess int
	totalSkipped int
	totalFailed  int
	totalBugs    int
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

	consecutiveFailures := 0
	startExp := monolith.LastExpedition + 1

	for exp := startExp; exp < startExp+p.config.MaxExpeditions; exp++ {
		select {
		case <-ctx.Done():
			LogWarn("%s", Msg("interrupted"))
			p.printSummary(exp - startExp)
			return 130
		default:
		}

		fmt.Println()
		LogExp("%s", fmt.Sprintf(Msg("departing"), exp))

		// === Rest at Flag: Lumina scan + Reserve recovery ===
		// (goroutine-parallel journal scan happens here)
		LogInfo("%s", Msg("resting_at_flag"))
		luminas := ScanJournalsForLumina(p.config.Continent)
		if len(luminas) > 0 {
			LogOK("%s", fmt.Sprintf(Msg("lumina_extracted"), len(luminas)))
			WriteLumina(p.config.Continent, luminas)
		}

		// Try to recover primary model
		p.reserve.TryRecoverPrimary()

		// Log current state
		LogInfo("%s", fmt.Sprintf(Msg("gradient_info"), p.gradient.FormatForPrompt()))
		LogInfo("%s", fmt.Sprintf(Msg("party_info"), p.reserve.Status()))

		expedition := &Expedition{
			Number:    exp,
			Continent: p.config.Continent,
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
			break
		}

		// === Send Expeditioner ===
		LogInfo("%s", fmt.Sprintf(Msg("sending"), p.reserve.ActiveModel()))

		output, err := expedition.Run(ctx)

		// === Process result ===
		if err != nil {
			if ctx.Err() == context.Canceled {
				LogWarn("%s", Msg("interrupted"))
				p.printSummary(exp - startExp + 1)
				return 130
			}

			LogError("%s", fmt.Sprintf(Msg("exp_failed"), exp, err))

			// If timeout, might be rate limit — consider switching to reserve
			if strings.Contains(err.Error(), "timeout") {
				p.reserve.ForceReserve()
			}

			p.gradient.Discharge()
			WriteFlag(p.config.Continent, exp, "error", "failed", monolith.Remaining)
			WriteJournal(p.config.Continent, &ExpeditionReport{
				Expedition: exp, IssueID: "?", IssueTitle: "?",
				MissionType: "?", Status: "failed", Reason: err.Error(),
				PRUrl: "none", BugIssues: "none",
			})
			consecutiveFailures++
			p.totalFailed++
		} else {
			report, status := ParseReport(output, exp)

			switch status {
			case StatusComplete:
				LogOK("%s", Msg("all_complete"))
				WriteFlag(p.config.Continent, exp, "all", "complete", "0")
				p.printSummary(exp - startExp + 1)
				return 0

			case StatusParseError:
				LogWarn("%s", Msg("report_parse_fail"))
				LogWarn("%s", fmt.Sprintf(Msg("output_check"), p.logDir, exp))
				p.gradient.Decay()
				consecutiveFailures++
				p.totalFailed++

			case StatusSuccess:
				p.handleSuccess(report)
				p.gradient.Charge()
				// Review gate: run code review on the PR if review command is configured
				if report.PRUrl != "" && report.PRUrl != "none" && p.config.ReviewCmd != "" {
					p.runReviewLoop(ctx, report)
				}
				WriteFlag(p.config.Continent, exp, report.IssueID, "success", report.Remaining)
				WriteJournal(p.config.Continent, report)
				consecutiveFailures = 0
				p.totalSuccess++

			case StatusSkipped:
				LogWarn("%s", fmt.Sprintf(Msg("issue_skipped"), report.IssueID, report.Reason))
				p.gradient.Decay()
				WriteFlag(p.config.Continent, exp, report.IssueID, "skipped", report.Remaining)
				WriteJournal(p.config.Continent, report)
				p.totalSkipped++

			case StatusFailed:
				LogError("%s", fmt.Sprintf(Msg("issue_failed"), report.IssueID, report.Reason))
				p.gradient.Discharge()
				WriteFlag(p.config.Continent, exp, report.IssueID, "failed", report.Remaining)
				WriteJournal(p.config.Continent, report)
				consecutiveFailures++
				p.totalFailed++
			}
		}

		// Gutter detection — Gommage
		if consecutiveFailures >= maxConsecutiveFailures {
			LogError("%s", fmt.Sprintf(Msg("gommage"), maxConsecutiveFailures))
			p.printSummary(exp - startExp + 1)
			return 1
		}

		// Return to base branch
		gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
		gitCmd.Dir = p.config.Continent
		_ = gitCmd.Run()

		// Cooldown
		LogInfo("%s", Msg("cooldown"))
		select {
		case <-time.After(10 * time.Second):
		case <-ctx.Done():
			LogWarn("%s", Msg("interrupted"))
			p.printSummary(exp - startExp + 1)
			return 130
		}
	}

	p.printSummary(p.config.MaxExpeditions)
	return 0
}

// runReviewLoop executes the code review command and, if comments are found,
// runs a lightweight Claude Code session to fix them. Repeats up to maxReviewCycles.
// Remaining review insights are appended to the report for journal recording.
// The entire loop is bounded by the expedition timeout to prevent hangs.
func (p *Paintress) runReviewLoop(ctx context.Context, report *ExpeditionReport) {
	// Bound the entire review loop with the expedition timeout
	timeout := time.Duration(p.config.TimeoutSec) * time.Second
	reviewCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for cycle := 1; cycle <= maxReviewCycles; cycle++ {
		// Check context before starting each cycle
		if reviewCtx.Err() != nil {
			LogWarn("%s", fmt.Sprintf(Msg("review_error"), reviewCtx.Err()))
			return
		}

		LogInfo("%s", fmt.Sprintf(Msg("review_running"), cycle, maxReviewCycles))

		result, err := RunReview(reviewCtx, p.config.ReviewCmd, p.config.Continent)
		if err != nil {
			LogWarn("%s", fmt.Sprintf(Msg("review_error"), err))
			return
		}

		if result.Passed {
			LogOK("%s", Msg("review_passed"))
			return
		}

		LogWarn("%s", fmt.Sprintf(Msg("review_comments"), cycle))

		if cycle >= maxReviewCycles {
			// Record remaining review insights in the report for journal
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Review not fully resolved: " + summarizeReview(result.Comments)
			LogWarn("%s", Msg("review_limit"))
			return
		}

		// Run a focused reviewfix session via Claude Code
		prompt := BuildReviewFixPrompt(report.Branch, result.Comments)

		claudeCmd := p.config.ClaudeCmd
		if claudeCmd == "" {
			claudeCmd = defaultClaudeCmd
		}

		model := p.reserve.ActiveModel()
		cmd := exec.CommandContext(reviewCtx, claudeCmd,
			"--model", model,
			"--continue",
			"--dangerously-skip-permissions",
			"--print",
			"-p", prompt,
		)
		cmd.Dir = p.config.Continent

		LogInfo("%s", fmt.Sprintf(Msg("reviewfix_running"), model))
		out, err := cmd.CombinedOutput()
		if err != nil {
			LogWarn("%s", fmt.Sprintf(Msg("reviewfix_error"), err))
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Reviewfix failed: " + summarizeReview(string(out))
			return
		}
	}
}

func (p *Paintress) handleSuccess(report *ExpeditionReport) {
	if report.MissionType == "verify" {
		LogQA("%s: %s", report.IssueID, report.IssueTitle)
		if report.BugsFound > 0 {
			LogQA("%s", fmt.Sprintf(Msg("qa_bugs"), report.BugsFound, report.BugIssues))
			p.totalBugs += report.BugsFound
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

func (p *Paintress) printSummary(expeditions int) {
	fmt.Println()
	fmt.Printf("%s╔══════════════════════════════════════════════╗%s\n", colorCyan, colorReset)
	fmt.Printf("%s║          The Paintress rests                 ║%s\n", colorCyan, colorReset)
	fmt.Printf("%s╚══════════════════════════════════════════════╝%s\n", colorCyan, colorReset)
	fmt.Println()
	LogInfo("%s", fmt.Sprintf(Msg("expeditions_sent"), expeditions))
	LogOK("%s", fmt.Sprintf(Msg("success_count"), p.totalSuccess))
	LogWarn("%s", fmt.Sprintf(Msg("skipped_count"), p.totalSkipped))
	LogError("%s", fmt.Sprintf(Msg("failed_count"), p.totalFailed))
	if p.totalBugs > 0 {
		LogQA("%s", fmt.Sprintf(Msg("bugs_count"), p.totalBugs))
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
