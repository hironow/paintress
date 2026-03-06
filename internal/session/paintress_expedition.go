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

// emitExpeditionCompleted emits an expedition.completed event via the emitter.
// OTel metric is recorded directly.
// Returns an error if event persistence fails — expedition completion is critical.
func (p *Paintress) emitExpeditionCompleted(exp int, status, issueID, bugsFound string) error {
	platform.RecordExpedition(context.Background(), status)
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
				attribute.String("model", model),
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
				rCtx, rCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer rCancel()
				if err := p.pool.Release(rCtx, workDir); err != nil {
					p.Logger.Warn("worktree release: %v", err)
				}
				workDir = ""
				relSpan.End()
			}
		}

		inboxDMails, scanErr := ScanInbox(p.config.Continent)
		if scanErr != nil {
			p.Logger.Warn("inbox scan for expedition #%d: %v", exp, scanErr)
		}
		for _, dm := range inboxDMails {
			if err := p.Emitter.EmitInboxReceived(dm.Name, dm.Severity, time.Now()); err != nil {
				p.Logger.Warn("inbox received event: %v", err)
			}
		}

		flagDir := workDir
		if flagDir == "" {
			flagDir = p.config.Continent
		}

		expedition := &Expedition{
			Number:      exp,
			Continent:   p.config.Continent,
			WorkDir:     workDir,
			Config:      p.config,
			LogDir:      p.logDir,
			Logger:      p.Logger,
			DataOut:     p.DataOut,
			ErrOut:      p.ErrOut,
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
				releaseWorkDir()
				expSpan.End()
				return nil
			}
			p.handleExpeditionError(expSpan, exp, expedition, flagDir, err)
		} else {
			if retErr := p.dispatchExpeditionResult(ctx, expCtx, expSpan, exp, expedition, flagDir, workDir, output, expStart); retErr != nil {
				releaseWorkDir()
				expSpan.End()
				return retErr
			}
		}

		if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) {
			p.stageEscalation(exp, maxConsecutiveFailures)
			expSpan.AddEvent("gommage",
				trace.WithAttributes(attribute.Int("consecutive_failures", maxConsecutiveFailures)),
			)
			releaseWorkDir()
			expSpan.End()
			p.Logger.Error("%s", fmt.Sprintf(domain.Msg("gommage"), maxConsecutiveFailures))
			if emitErr := p.Emitter.EmitGommage(exp, time.Now()); emitErr != nil {
				p.Logger.Error("gommage event lost: %v", emitErr)
			}
			return errGommage
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
		case <-time.After(10 * time.Second):
		case <-ctx.Done():
			return nil
		}
	}
}

// handleExpeditionError processes a failed expedition run: switches model on
// timeout, discharges gradient, writes flag/journal, and updates counters.
func (p *Paintress) handleExpeditionError(expSpan trace.Span, exp int, expedition *Expedition, flagDir string, runErr error) {
	p.Logger.Error("%s", fmt.Sprintf(domain.Msg("exp_failed"), exp, runErr))
	if strings.Contains(runErr.Error(), "timeout") {
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
	if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "discharge", time.Now()); err != nil {
		p.Logger.Warn("gradient event: %v", err)
	}

	midHighNames := expedition.MidHighSeverityDMails()
	midHighCount := len(midHighNames)
	if midHighCount > 0 {
		p.totalMidHighSeverity.Add(int64(midHighCount))
	}

	p.writeFlag(flagDir, exp, "error", "failed", "?", midHighCount)
	errReport := &domain.ExpeditionReport{
		Expedition: exp, IssueID: "?", IssueTitle: "?",
		MissionType: "?", Status: "failed", Reason: runErr.Error(),
		FailureType: "blocker",
		PRUrl:       "none", BugIssues: "none",
	}
	if midHighCount > 0 {
		errReport.HighSeverityDMails = strings.Join(midHighNames, ", ")
	}
	WriteJournal(p.config.Continent, errReport)
	if err := p.emitExpeditionCompleted(exp, "failed", "", ""); err != nil {
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
func (p *Paintress) dispatchExpeditionResult(ctx context.Context, expCtx context.Context, expSpan trace.Span, exp int, expedition *Expedition, flagDir, workDir, output string, expStart time.Time) error {
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
		if err := p.emitExpeditionCompleted(exp, "parse_error", "", ""); err != nil {
			p.Logger.Error("expedition completion event lost: %v", err)
		}
		p.consecutiveFailures.Add(1)
		p.totalFailed.Add(1)
	case domain.StatusSuccess:
		expSpan.AddEvent("expedition.complete",
			trace.WithAttributes(
				attribute.String("status", "success"),
				attribute.String("issue_id", report.IssueID),
				attribute.String("mission_type", report.MissionType),
			),
		)
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
		if report.PRUrl != "" && report.PRUrl != "none" && p.config.ReviewCmd != "" {
			totalTimeout := time.Duration(p.config.TimeoutSec) * time.Second
			remaining := totalTimeout - time.Since(expStart)
			if remaining > 0 {
				p.runReviewLoop(ctx, report, remaining, workDir)
			}
		}
		p.writeFlag(flagDir, exp, report.IssueID, "success", report.Remaining, midHighCount)
		WriteJournal(p.config.Continent, report)
		if err := p.emitExpeditionCompleted(exp, "success", report.IssueID, fmt.Sprintf("%d", report.BugsFound)); err != nil {
			p.Logger.Error("expedition completion event lost: %v", err)
		}
		if dm := domain.NewReportDMail(report); dm.Name != "" {
			if err := SendDMail(p.outboxStore, dm, p.Emitter); err != nil {
				p.Logger.Warn("dmail send: %v", err)
			}
		}
		for _, dm := range expedition.InboxDMails {
			if err := ArchiveInboxDMail(p.config.Continent, dm.Name, p.Emitter); err != nil {
				p.Logger.Warn("dmail archive: %v", err)
			}
		}
		p.consecutiveFailures.Store(0)
		p.totalSuccess.Add(1)
	case domain.StatusSkipped:
		p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("issue_skipped"), report.IssueID, report.Reason))
		p.gradient.Decay()
		if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "decay", time.Now()); err != nil {
			p.Logger.Warn("gradient event: %v", err)
		}
		p.writeFlag(flagDir, exp, report.IssueID, "skipped", report.Remaining, midHighCount)
		WriteJournal(p.config.Continent, report)
		if err := p.emitExpeditionCompleted(exp, "skipped", report.IssueID, ""); err != nil {
			p.Logger.Error("expedition completion event lost: %v", err)
		}
		p.totalSkipped.Add(1)
	case domain.StatusFailed:
		p.Logger.Error("%s", fmt.Sprintf(domain.Msg("issue_failed"), report.IssueID, report.Reason))
		p.gradient.Discharge()
		if err := p.Emitter.EmitGradientChange(p.gradient.Level(), "discharge", time.Now()); err != nil {
			p.Logger.Warn("gradient event: %v", err)
		}
		p.writeFlag(flagDir, exp, report.IssueID, "failed", report.Remaining, midHighCount)
		WriteJournal(p.config.Continent, report)
		if err := p.emitExpeditionCompleted(exp, "failed", report.IssueID, ""); err != nil {
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
