package session

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// expeditionCooldown is the pause between expedition cycles.
// Declared as var (not const) so tests can shorten it.
var expeditionCooldown = 10 * time.Second

// worktreeReleaseTimeout is the per-call timeout for worktree release operations.
var worktreeReleaseTimeout = 10 * time.Second

// emitExpeditionCompleted emits an expedition.completed event via the emitter.
// OTel metric is recorded directly with model attribution.
// Returns an error if event persistence fails — expedition completion is critical.
func (p *Paintress) emitExpeditionCompleted(ctx context.Context, exp int, status, issueID, bugsFound, model string) error {
	platform.RecordExpedition(ctx, status, model)
	return p.Emitter.EmitCompleteExpedition(exp, status, issueID, bugsFound, time.Now())
}

func (p *Paintress) runWorker(ctx context.Context, workerID int, startExp int, luminas []domain.Lumina) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		exp := int(p.expCounter.Add(1) - 1)
		if exp >= startExp+p.config.MaxExpeditions {
			return nil
		}

		p.totalAttempted.Add(1)
		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("departing"), exp))
		p.reserve.TryRecoverPrimary()
		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("gradient_info"), p.gradient.FormatForPrompt()))
		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("party_info"), p.reserve.Status()))

		model := p.reserve.ActiveModel()
		if err := p.Emitter.EmitStartExpedition(exp, workerID, model, time.Now()); err != nil {
			p.Logger.Warn("start expedition event: %v", err)
		}
		expCtx, expSpan := platform.Tracer.Start(ctx, "expedition", // nosemgrep: adr0003-otel-span-without-defer-end — expSpan.End() called after expedition loop body [permanent]
			trace.WithAttributes(
				attribute.Int("expedition.number", exp),
				attribute.Int("worker.id", workerID),
				attribute.String("model", platform.SanitizeUTF8(model)),
			),
		)

		var workDir string
		if p.pool != nil {
			_, acqSpan := platform.Tracer.Start(expCtx, "worktree.acquire") // nosemgrep: adr0003-otel-span-without-defer-end — acqSpan.End() called immediately after Acquire() [permanent]
			workDir = p.pool.Acquire()
			acqSpan.End()
		}
		releaseWorkDir := func() {
			if p.pool != nil && workDir != "" {
				_, relSpan := platform.Tracer.Start(expCtx, "worktree.release") // nosemgrep: adr0003-otel-span-without-defer-end — relSpan.End() called after Release() [permanent]
				rCtx, rCancel := context.WithTimeout(context.Background(), worktreeReleaseTimeout)
				defer rCancel()
				if err := p.pool.Release(rCtx, workDir); err != nil {
					p.Logger.Warn("worktree release: %v", err)
				}
				workDir = ""
				relSpan.End()
			}
		}

		inboxDMails, scanErr := ScanInbox(expCtx, p.config.Continent)
		if scanErr != nil {
			p.Logger.Warn("inbox scan for expedition #%d: %v", exp, scanErr)
		}
		for _, dm := range inboxDMails {
			domain.LogBanner(p.Logger, domain.BannerRecv, dm.Kind, dm.Name, dm.Description)
			if err := p.Emitter.EmitInboxReceived(dm.Name, dm.Severity, time.Now()); err != nil {
				p.Logger.Warn("inbox received event: %v", err)
			}
		}

		// Pre-flight triage: process action fields before expedition creation.
		// escalate/resolve D-Mails are handled immediately and removed;
		// only pass-through D-Mails reach the expedition prompt.
		inboxDMails = p.triagePreFlightDMails(expCtx, inboxDMails)

		flagDir := workDir
		if flagDir == "" {
			flagDir = p.config.Continent
		}

		// Wave mode: fetch pending step targets from archive projection
		var waveTarget *domain.ExpeditionTarget
		if p.trackingMode.IsWave() && p.targetProvider != nil {
			targets, tErr := p.targetProvider.FetchTargets(expCtx)
			if tErr != nil {
				p.Logger.Warn("wave target fetch: %v", tErr)
			} else if len(targets) == 0 {
				p.Logger.Info("Wave mode: no pending targets, expedition cycle complete")
				expSpan.End()
				return nil
			} else {
				// Pick first pending target; claim it before expedition starts
				target := targets[0]
				if p.claimRegistry != nil {
					if ok, holder := p.claimRegistry.TryClaim(target.ID, exp); !ok {
						p.Logger.Info("Wave target %s claimed by expedition #%d, trying next", target.ID, holder)
						// Try remaining targets
						claimed := false
						for _, t := range targets[1:] {
							if ok2, _ := p.claimRegistry.TryClaim(t.ID, exp); ok2 {
								target = t
								claimed = true
								break
							}
						}
						if !claimed {
							p.Logger.Info("All wave targets claimed, skipping expedition #%d", exp)
							p.totalSkipped.Add(1)
							expSpan.End()
							continue
						}
					}
				}
				waveTarget = &target
				p.Logger.Info("Wave target: %s — %s", target.ID, target.Title)
			}
		}

		expedition := &Expedition{
			Number:        exp,
			Continent:     p.config.Continent,
			WorkDir:       workDir,
			Config:        p.config,
			LogDir:        p.logDir,
			Logger:        p.Logger,
			DataOut:       p.DataOut,
			ErrOut:        p.ErrOut,
			Luminas:       luminas,
			Gradient:      p.gradient,
			Reserve:       p.reserve,
			InboxDMails:   inboxDMails,
			Notifier:      p.notifier,
			ClaimRegistry: p.claimRegistry,
			Target:        waveTarget,
		}
		// Wave mode: pre-set the claim key from target ID (no flag watcher needed)
		if waveTarget != nil {
			expedition.setCurrentIssue(waveTarget.ID)
		}

		// Wrap releaseWorkDir to also release the issue claim.
		origReleaseWorkDir := releaseWorkDir
		releaseWorkDir = func() {
			expedition.ReleaseClaim()
			origReleaseWorkDir()
		}

		// archiveInbox moves all inbox D-Mails to archive. Called on error/gommage
		// paths; the success path is covered by dispatchExpeditionResult's defer.
		archiveInbox := func() {
			for _, dm := range expedition.InboxDMails {
				if err := ArchiveInboxDMail(ctx, p.config.Continent, dm.Name, p.Emitter); err != nil {
					p.Logger.Warn("dmail archive: %v", err)
				}
			}
		}

		if p.config.DryRun {
			promptFile := filepath.Join(p.logDir, fmt.Sprintf("expedition-%03d-prompt.md", exp))
			if err := os.WriteFile(promptFile, []byte(expedition.BuildPrompt()), 0644); err != nil {
				p.Logger.Error("failed to write dry-run prompt: %v", err)
				releaseWorkDir()
				expSpan.End()
				continue
			}
			p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("dry_run_prompt"), promptFile))
			p.totalSuccess.Add(1)
			releaseWorkDir()
			expSpan.End()
			continue
		}

		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("sending"), p.reserve.ActiveModel()))
		expStart := time.Now()
		output, err := expedition.Run(expCtx)

		if err != nil {
			if ctx.Err() != nil {
				archiveInbox()
				releaseWorkDir()
				expSpan.End()
				return nil
			}
			p.handleExpeditionError(expCtx, expSpan, exp, expedition, flagDir, model, err)
			archiveInbox()
		} else {
			if retErr := p.dispatchExpeditionResult(ctx, expCtx, expSpan, exp, expedition, flagDir, workDir, output, model, expStart); retErr != nil {
				releaseWorkDir()
				expSpan.End()
				return retErr
			}
		}

		if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) && p.escalationFired.CompareAndSwap(false, true) {
			reasons := recentFailureReasons(p.config.Continent, 5)
			decision := p.recoveryDecider.DecideRecovery(reasons)

			// Best-effort: write Gommage insight for cross-tool observability
			gommageWriter := NewInsightWriter(
				domain.InsightsDir(p.config.Continent),
				domain.RunDir(p.config.Continent),
			)
			WriteGommageInsight(gommageWriter, exp, maxConsecutiveFailures, p.config.Continent, decision.Class)

			if emitErr := p.Emitter.EmitGommage(exp, time.Now()); emitErr != nil {
				p.Logger.Error("gommage event lost: %v", emitErr)
			}

			expSpan.AddEvent("gommage",
				trace.WithAttributes(
					attribute.Int("consecutive_failures", maxConsecutiveFailures),
					attribute.String("gommage.class", platform.SanitizeUTF8(string(decision.Class))),
					attribute.String("gommage.action", platform.SanitizeUTF8(string(decision.RecoveryKind))),
					attribute.Int("gommage.retry_num", decision.RetryNum),
				),
			)

			if p.executeRecovery(ctx, decision, exp, expedition) {
				// Recovery says retry: save checkpoint, keep worktree, reset counter
				p.saveCheckpoint(exp, CheckpointSubprocessStart, workDir)
				p.consecutiveFailures.Store(0)
				p.escalationFired.Store(false)
				expSpan.End()
				continue // retry same issue
			}

			// Halt path (unchanged behavior)
			p.stageEscalation(ctx, exp, maxConsecutiveFailures)
			releaseWorkDir()
			expSpan.End()
			p.Logger.Error("%s", fmt.Sprintf(domain.Msg("gommage"), maxConsecutiveFailures))
			return errGommage
		}

		if p.consecutiveSkips.Load() >= int64(maxConsecutiveSkips) {
			expSpan.AddEvent("all_skipped",
				trace.WithAttributes(attribute.Int("consecutive_skips", maxConsecutiveSkips)),
			)
			releaseWorkDir()
			expSpan.End()
			p.Logger.Warn("all expeditions skipped %d times consecutively — no actionable work available", maxConsecutiveSkips)
			return errAllSkipped
		}

		releaseWorkDir()
		expSpan.End()
		if p.pool == nil {
			gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
			gitCmd.Dir = p.config.Continent
			_ = gitCmd.Run()
		}

		p.Logger.Info("%s", domain.Msg("cooldown"))
		select {
		case <-time.After(expeditionCooldown):
		case <-ctx.Done():
			return nil
		}
	}
}

// handleExpeditionError processes a failed expedition run: switches model on
// timeout, discharges gradient, writes flag/journal, and updates counters.
func (p *Paintress) handleExpeditionError(expCtx context.Context, expSpan trace.Span, exp int, expedition *Expedition, flagDir, model string, runErr error) {
	p.Logger.Error("%s", fmt.Sprintf(domain.Msg("exp_failed"), exp, runErr))
	if strings.Contains(runErr.Error(), "timeout") {
		prevModel := p.reserve.ActiveModel()
		p.reserve.ForceReserve()
		newModel := p.reserve.ActiveModel()
		if newModel != prevModel {
			expSpan.AddEvent("model.switched",
				trace.WithAttributes(
					attribute.String("from", platform.SanitizeUTF8(prevModel)),
					attribute.String("to", platform.SanitizeUTF8(newModel)),
				),
			)
		}
	}
	p.gradient.Discharge()
	if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "discharge", time.Now()); err != nil {
		p.Logger.Warn("gradient event: %v", err)
	}

	midHighNames := expedition.MidHighSeverityDMails()
	midHighCount := len(midHighNames)
	if midHighCount > 0 {
		p.totalMidHighSeverity.Add(int64(midHighCount))
	}

	// Inject rate_limit marker if reserve model is active (enables ClassifyGommage detection)
	reason := runErr.Error()
	if p.reserve.IsOnReserve() && !strings.Contains(reason, "timeout") {
		reason = "rate_limit: " + reason
	}

	p.writeFlag(flagDir, exp, "error", "failed", "?", midHighCount)
	errReport := &domain.ExpeditionReport{
		Expedition: exp, IssueID: "?", IssueTitle: "?",
		MissionType: "?", Status: "failed", Reason: reason,
		FailureType: "blocker",
		PRUrl:       "none", BugIssues: "none",
	}
	if midHighCount > 0 {
		errReport.HighSeverityDMails = strings.Join(midHighNames, ", ")
	}
	WriteJournal(p.config.Continent, errReport)
	if err := p.emitExpeditionCompleted(expCtx, exp, "failed", "", "", model); err != nil {
		p.Logger.Error("expedition completion event lost: %v", err)
	}
	if midHighCount > 0 {
		p.Logger.Warn("Expedition #%d: %d HIGH severity D-Mail received mid-expedition", exp, midHighCount)
	}
	p.consecutiveFailures.Add(1)
	p.totalFailed.Add(1)
}

// dispatchExpeditionResult parses the expedition output, dispatches based on
// status (complete/success/skipped/failed/parse-error), and updates counters.
// Returns errComplete when all issues are done; nil otherwise.
func (p *Paintress) dispatchExpeditionResult(ctx context.Context, expCtx context.Context, expSpan trace.Span, exp int, expedition *Expedition, flagDir, workDir, output, model string, expStart time.Time) error {
	// Archive ALL inbox D-Mails when this function returns, regardless of status.
	// Without this, D-Mails remain in inbox and re-trigger waiting mode infinitely.
	defer func() {
		for _, dm := range expedition.InboxDMails {
			if err := ArchiveInboxDMail(ctx, p.config.Continent, dm.Name, p.Emitter); err != nil {
				p.Logger.Warn("dmail archive: %v", err)
			}
		}
	}()

	_, parseSpan := platform.Tracer.Start(expCtx, "report.parse") // nosemgrep: adr0003-otel-span-without-defer-end -- End() called immediately after ParseReport [permanent]
	report, status := domain.ParseReport(output, exp)
	parseSpan.End()

	midHighNames := expedition.MidHighSeverityDMails()
	midHighCount := len(midHighNames)
	if midHighCount > 0 {
		if report != nil {
			report.HighSeverityDMails = strings.Join(midHighNames, ", ")
		}
		p.totalMidHighSeverity.Add(int64(midHighCount))
	}

	switch status {
	case domain.StatusComplete:
		expSpan.AddEvent("expedition.complete",
			trace.WithAttributes(attribute.String("status", "all_complete")),
		)
		p.writeFlag(flagDir, exp, "all", "complete", "0", midHighCount)
		p.Logger.OK("%s", domain.Msg("all_complete"))
		return errComplete
	case domain.StatusParseError:
		p.Logger.Warn("%s", domain.Msg("report_parse_fail"))
		p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("output_check"), p.logDir, exp))
		p.gradient.Decay()
		if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "decay", time.Now()); err != nil {
			p.Logger.Warn("gradient event: %v", err)
		}
		p.writeFlag(flagDir, exp, "?", "parse_error", "?", midHighCount)
		parseErrReport := &domain.ExpeditionReport{
			Expedition: exp, IssueID: "?", IssueTitle: "?",
			MissionType: "?", Status: "parse_error", Reason: "report markers not found",
			FailureType: "blocker",
			PRUrl:       "none", BugIssues: "none",
		}
		if midHighCount > 0 {
			parseErrReport.HighSeverityDMails = strings.Join(midHighNames, ", ")
		}
		WriteJournal(p.config.Continent, parseErrReport)
		if err := p.emitExpeditionCompleted(expCtx, exp, "parse_error", "", "", model); err != nil {
			p.Logger.Error("expedition completion event lost: %v", err)
		}
		p.consecutiveFailures.Add(1)
		p.totalFailed.Add(1)
	case domain.StatusSuccess:
		expSpan.AddEvent("expedition.complete",
			trace.WithAttributes(
				attribute.String("status", "success"),
				attribute.String("issue_id", platform.SanitizeUTF8(report.IssueID)),
				attribute.String("mission_type", platform.SanitizeUTF8(report.MissionType)),
			),
		)
		// Wave mode: transfer wave/step context to report for D-Mail projection
		if expedition.Target != nil {
			report.WaveID = expedition.Target.WaveID
			report.StepID = expedition.Target.StepID
		}
		p.handleSuccess(report)
		p.gradient.Charge()
		if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "charge", time.Now()); err != nil {
			p.Logger.Warn("gradient event: %v", err)
		}
		if matched := expedition.MidMatchedDMails(); len(matched) > 0 {
			totalTimeout := time.Duration(p.config.TimeoutSec) * time.Second
			followUpBudget := totalTimeout - time.Since(expStart)
			for _, dm := range matched {
				if dm.Action != "" {
					p.handleFeedbackAction(ctx, dm, workDir, followUpBudget)
				} else {
					p.runFollowUp(ctx, []domain.DMail{dm}, workDir, followUpBudget)
				}
			}
		}
		var reviewStatus domain.ReviewGateStatus
		if report.PRUrl != "" && report.PRUrl != "none" && p.config.ReviewCmd != "" {
			totalTimeout := time.Duration(p.config.TimeoutSec) * time.Second
			remaining := totalTimeout - time.Since(expStart)
			if remaining > 0 {
				reviewStatus = p.runReviewLoop(ctx, report, remaining, workDir)
			}
		} else if report.PRUrl != "" && report.PRUrl != "none" {
			reviewStatus = domain.ReviewGateStatus{Skipped: true}
		}
		if report.PRUrl != "" && report.PRUrl != "none" {
			if err := UpdatePRReviewGate(ctx, report.PRUrl, reviewStatus, p.Logger); err != nil {
				p.Logger.Warn("PR review gate update: %v", err)
			}
		}
		p.writeFlag(flagDir, exp, report.IssueID, "success", report.Remaining, midHighCount)
		WriteJournal(p.config.Continent, report)
		if err := WritePRIndex(p.config.Continent, report); err != nil {
			p.Logger.Warn("pr index: %v", err)
		}
		if err := p.emitExpeditionCompleted(expCtx, exp, "success", report.IssueID, fmt.Sprintf("%d", report.BugsFound), model); err != nil {
			p.Logger.Error("expedition completion event lost: %v", err)
		}
		if dm := domain.NewReportDMail(report, p.gradient.Level()); dm.Name != "" {
			domain.LogBanner(p.Logger, domain.BannerSend, dm.Kind, dm.Name, dm.Description)
			if err := SendDMail(ctx, p.outboxStore, dm, p.Emitter); err != nil {
				p.Logger.Warn("dmail send: %v", err)
			}
		}
		p.consecutiveFailures.Store(0)
		p.consecutiveSkips.Store(0)
		p.escalationFired.Store(false)
		p.recoveryDecider.ResetRecovery()
		p.totalSuccess.Add(1)
	case domain.StatusSkipped:
		p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("issue_skipped"), report.IssueID, report.Reason))
		p.gradient.Decay()
		if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "decay", time.Now()); err != nil {
			p.Logger.Warn("gradient event: %v", err)
		}
		p.writeFlag(flagDir, exp, report.IssueID, "skipped", report.Remaining, midHighCount)
		WriteJournal(p.config.Continent, report)
		if err := p.emitExpeditionCompleted(expCtx, exp, "skipped", report.IssueID, "", model); err != nil {
			p.Logger.Error("expedition completion event lost: %v", err)
		}
		// Re-review past PRs when skipped and review_cmd is configured.
		if p.config.ReviewCmd != "" {
			p.runSkipReview(ctx, workDir, expStart)
		}
		p.consecutiveSkips.Add(1)
		p.totalSkipped.Add(1)
	case domain.StatusFailed:
		p.Logger.Error("%s", fmt.Sprintf(domain.Msg("issue_failed"), report.IssueID, report.Reason))
		p.gradient.Discharge()
		if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "discharge", time.Now()); err != nil {
			p.Logger.Warn("gradient event: %v", err)
		}
		p.writeFlag(flagDir, exp, report.IssueID, "failed", report.Remaining, midHighCount)
		WriteJournal(p.config.Continent, report)
		if err := p.emitExpeditionCompleted(expCtx, exp, "failed", report.IssueID, "", model); err != nil {
			p.Logger.Error("expedition completion event lost: %v", err)
		}
		p.consecutiveFailures.Add(1)
		p.totalFailed.Add(1)
	}

	if midHighCount > 0 {
		p.Logger.Warn("Expedition #%d: %d HIGH severity D-Mail received mid-expedition", exp, midHighCount)
	}
	return nil
}

func (p *Paintress) handleSuccess(report *domain.ExpeditionReport) {
	if report.MissionType == "verify" {
		p.Logger.Info("%s: %s", report.IssueID, report.IssueTitle)
		if report.BugsFound > 0 {
			p.Logger.Info("%s", fmt.Sprintf(domain.Msg("qa_bugs"), report.BugsFound, report.BugIssues))
			p.totalBugs.Add(int64(report.BugsFound))
		} else {
			p.Logger.Info("%s", domain.Msg("qa_all_pass"))
		}
	} else {
		p.Logger.OK("%s: %s [%s]", report.IssueID, report.IssueTitle, report.MissionType)
	}
	if report.PRUrl != "" && report.PRUrl != "none" {
		p.Logger.OK("PR: %s", report.PRUrl)
	}
	if report.Remaining != "" {
		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("monolith_reads"), report.Remaining))
	}
}
