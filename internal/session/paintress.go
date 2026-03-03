package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hironow/paintress"
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
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
	config      paintress.Config
	logDir      string
	Logger      *domain.Logger
	DataOut     io.Writer // stdout-equivalent for data output
	StdinIn     io.Reader // stdin-equivalent for interactive input
	devServer   *DevServer
	gradient    *paintress.GradientGauge
	reserve     *domain.ReserveParty
	pool        *WorktreePool // nil when --workers=0
	notifier    paintress.Notifier
	approver    paintress.Approver
	outboxStore paintress.OutboxStore  // transactional outbox for D-Mail delivery
	eventStore  domain.EventStore      // append-only event log (fire-and-forget)
	Dispatcher  domain.EventDispatcher // policy engine (best-effort dispatch after emit)

	// Retry tracking: maps sorted issue keys to attempt count
	retryMu      sync.Mutex
	retryTracker map[string]int

	// Swarm Mode: atomic counters for concurrent worker access
	expCounter           atomic.Int64
	totalAttempted       atomic.Int64
	totalSuccess         atomic.Int64
	totalSkipped         atomic.Int64
	totalFailed          atomic.Int64
	totalBugs            atomic.Int64
	totalMidHighSeverity atomic.Int64
	consecutiveFailures  atomic.Int64
}

// emitEvent appends a single event to the event store. Errors are logged
// but never propagated — the event log is observational, not critical.
// After persistence, the event is dispatched to the PolicyEngine best-effort.
func (p *Paintress) emitEvent(eventType domain.EventType, data any) {
	// Record OTel metric for expedition completions (fire-and-forget, independent of event store)
	if eventType == domain.EventExpeditionCompleted {
		if d, ok := data.(domain.ExpeditionCompletedData); ok {
			platform.RecordExpedition(context.Background(), d.Status)
		}
	}
	if p.eventStore == nil {
		return
	}
	ev, err := domain.NewEvent(eventType, data, time.Now())
	if err != nil {
		p.Logger.Debug("event emit marshal: %v", err)
		return
	}
	if err := p.eventStore.Append(ev); err != nil {
		p.Logger.Debug("event emit append: %v", err)
		platform.RecordEventEmitError(context.Background(), string(eventType))
	}
	// Best-effort policy dispatch
	if p.Dispatcher != nil {
		if dispatchErr := p.Dispatcher.Dispatch(context.Background(), ev); dispatchErr != nil {
			p.Logger.Debug("policy dispatch %s: %v", eventType, dispatchErr)
		}
	}
}

func NewPaintress(cfg paintress.Config, logger *domain.Logger, dataOut io.Writer, stdinIn io.Reader, eventStore domain.EventStore) *Paintress {
	if logger == nil {
		logger = domain.NewLogger(nil, false)
	}
	if dataOut == nil {
		dataOut = io.Discard
	}
	if stdinIn == nil {
		stdinIn = strings.NewReader("")
	}
	logDir := filepath.Join(cfg.Continent, ".expedition", ".run", "logs")
	os.MkdirAll(logDir, 0755)

	// Reserve Party: parse model string for reserves
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
	var notifier paintress.Notifier
	if cfg.NotifyCmd != "" {
		notifier = NewCmdNotifier(cfg.NotifyCmd)
	} else {
		notifier = &LocalNotifier{}
	}

	// Wire approver based on config
	var approver paintress.Approver
	switch {
	case cfg.AutoApprove:
		approver = &paintress.AutoApprover{}
	case cfg.ApproveCmd != "":
		approver = NewCmdApprover(cfg.ApproveCmd)
	default:
		promptOut := logger.Writer()
		if promptOut == io.Discard || promptOut == dataOut {
			promptOut = os.Stderr // nosemgrep: adr0002-no-os-stderr-in-internal — fallback for quiet-mode + interactive approval; cmd layer cannot predict this condition
		}
		approver = NewStdinApprover(stdinIn, promptOut)
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	cfgCopy := cfg
	cfgCopy.MaxRetries = maxRetries

	p := &Paintress{
		config:       cfgCopy,
		logDir:       logDir,
		Logger:       logger,
		DataOut:      dataOut,
		StdinIn:      stdinIn,
		gradient:     paintress.NewGradientGauge(gradientMax),
		reserve:      domain.NewReserveParty(primary, reserves, logger),
		notifier:     notifier,
		approver:     approver,
		eventStore:   eventStore,
		retryTracker: make(map[string]int),
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
	ctx, rootSpan := platform.Tracer.Start(ctx, "paintress.run",
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
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		p.Logger.Warn("log file: %v", err)
	} else {
		p.Logger.SetExtraWriter(logFile)
		defer logFile.Close()
	}

	// Initialize transactional outbox store for D-Mail delivery.
	outboxStore, err := NewOutboxStoreForContinent(p.config.Continent)
	if err != nil {
		p.Logger.Error("outbox store: %v", err)
		rootSpan.End()
		return 1
	}
	defer outboxStore.Close()
	p.outboxStore = outboxStore

	p.printBanner()
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("continent"), p.config.Continent))

	// Start dev server (stays alive across expeditions)
	if !p.config.DryRun && p.devServer != nil {
		if err := p.devServer.Start(ctx); err != nil {
			p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("devserver_warn"), err))
		}
		defer p.devServer.Stop()
	}

	// Initialize worktree pool if workers > 0.
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

	monolith := reconcileFlags(p.config.Continent, p.config.Workers)
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("monolith_reads"), monolith.Remaining))
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("max_expeditions"), p.config.MaxExpeditions))
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("party_info"), p.reserve.Status()))
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("gradient_info"), p.gradient.FormatForPrompt()))
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("timeout_info"), p.config.TimeoutSec))
	claudeCmd := p.config.ClaudeCmd
	if claudeCmd == "" {
		claudeCmd = paintress.DefaultClaudeCmd
	}
	if claudeCmd != paintress.DefaultClaudeCmd {
		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("claude_cmd_info"), claudeCmd))
	}
	if p.config.DryRun {
		p.Logger.Warn("%s", paintress.Msg("dry_run"))
	}
	fmt.Fprintln(p.Logger.Writer())

	// === Swarm Mode: reset run-scoped counters and launch workers ===
	p.totalAttempted.Store(0)
	p.totalSuccess.Store(0)
	p.totalSkipped.Store(0)
	p.totalFailed.Store(0)
	p.totalBugs.Store(0)
	p.totalMidHighSeverity.Store(0)
	p.consecutiveFailures.Store(0)

	startExp := monolith.LastExpedition + 1
	p.expCounter.Store(int64(startExp))

	// Pre-flight Lumina scan (once, before workers start)
	luminas := ScanJournalsForLumina(p.config.Continent)
	if len(luminas) > 0 {
		p.Logger.OK("%s", fmt.Sprintf(paintress.Msg("lumina_extracted"), len(luminas)))
	}

	// Pre-flight HIGH severity gate (once, before workers start).
	preflightInbox, scanErr := ScanInbox(p.config.Continent)
	if scanErr != nil {
		p.Logger.Error("inbox scan failed (fail-closed): %v", scanErr)
		return 1
	}
	if highDMails := paintress.FilterHighSeverity(preflightInbox); len(highDMails) > 0 {
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
			p.Logger.Error("approval request failed (fail-closed): %v", err)
			return 1
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

	err = g.Wait()

	// Consolidate: write the latest checkpoint back to Continent
	if latest := reconcileFlags(p.config.Continent, p.config.Workers); latest.LastExpedition > 0 {
		if flagErr := WriteFlag(p.config.Continent, latest.LastExpedition, latest.LastIssue,
			latest.LastStatus, latest.Remaining, latest.MidHighSeverity); flagErr != nil {
			p.Logger.Warn("consolidate flag: %v", flagErr)
		}
	}

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

func (p *Paintress) runWorker(ctx context.Context, workerID int, startExp int, luminas []paintress.Lumina) error {
	for {
		if ctx.Err() != nil {
			return nil
		}

		exp := int(p.expCounter.Add(1) - 1)
		if exp >= startExp+p.config.MaxExpeditions {
			return nil
		}

		p.totalAttempted.Add(1)
		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("departing"), exp))
		p.reserve.TryRecoverPrimary()
		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("gradient_info"), p.gradient.FormatForPrompt()))
		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("party_info"), p.reserve.Status()))

		model := p.reserve.ActiveModel()
		p.emitEvent(domain.EventExpeditionStarted, domain.ExpeditionStartedData{
			Expedition: exp, Worker: workerID, Model: model,
		})
		expCtx, expSpan := platform.Tracer.Start(ctx, "expedition",
			trace.WithAttributes(
				attribute.Int("expedition.number", exp),
				attribute.Int("worker.id", workerID),
				attribute.String("model", model),
			),
		)

		var workDir string
		if p.pool != nil {
			_, acqSpan := platform.Tracer.Start(expCtx, "worktree.acquire")
			workDir = p.pool.Acquire()
			acqSpan.End()
		}
		releaseWorkDir := func() {
			if p.pool != nil && workDir != "" {
				_, relSpan := platform.Tracer.Start(expCtx, "worktree.release")
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
			p.emitEvent(domain.EventInboxReceived, domain.InboxReceivedData{
				Name: dm.Name, Severity: dm.Severity,
			})
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
			p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("dry_run_prompt"), promptFile))
			p.totalSuccess.Add(1)
			releaseWorkDir()
			expSpan.End()
			continue
		}

		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("sending"), p.reserve.ActiveModel()))
		expStart := time.Now()
		output, err := expedition.Run(expCtx)

		if err != nil {
			if ctx.Err() != nil {
				releaseWorkDir()
				expSpan.End()
				return nil
			}
			p.Logger.Error("%s", fmt.Sprintf(paintress.Msg("exp_failed"), exp, err))
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
			p.emitEvent(domain.EventGradientChanged, domain.GradientChangedData{
				Level: p.gradient.Level(), Operator: "discharge",
			})

			midHighNames := expedition.MidHighSeverityDMails()
			midHighCount := len(midHighNames)
			if midHighCount > 0 {
				p.totalMidHighSeverity.Add(int64(midHighCount))
			}

			p.writeFlag(flagDir, exp, "error", "failed", "?", midHighCount)
			errReport := &paintress.ExpeditionReport{
				Expedition: exp, IssueID: "?", IssueTitle: "?",
				MissionType: "?", Status: "failed", Reason: err.Error(),
				FailureType: "blocker",
				PRUrl:       "none", BugIssues: "none",
			}
			if midHighCount > 0 {
				errReport.HighSeverityDMails = strings.Join(midHighNames, ", ")
			}
			WriteJournal(p.config.Continent, errReport)
			p.emitEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
				Expedition: exp, Status: "failed",
			})
			if midHighCount > 0 {
				p.Logger.Warn("Expedition #%d: %d HIGH severity D-Mail received mid-expedition", exp, midHighCount)
			}
			p.consecutiveFailures.Add(1)
			p.totalFailed.Add(1)
		} else {
			_, parseSpan := platform.Tracer.Start(expCtx, "report.parse")
			report, status := paintress.ParseReport(output, exp)
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
			case paintress.StatusComplete:
				expSpan.AddEvent("expedition.complete",
					trace.WithAttributes(attribute.String("status", "all_complete")),
				)
				p.writeFlag(flagDir, exp, "all", "complete", "0", midHighCount)
				releaseWorkDir()
				expSpan.End()
				p.Logger.OK("%s", paintress.Msg("all_complete"))
				return errComplete
			case paintress.StatusParseError:
				p.Logger.Warn("%s", paintress.Msg("report_parse_fail"))
				p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("output_check"), p.logDir, exp))
				p.gradient.Decay()
				p.emitEvent(domain.EventGradientChanged, domain.GradientChangedData{
					Level: p.gradient.Level(), Operator: "decay",
				})
				p.writeFlag(flagDir, exp, "?", "parse_error", "?", midHighCount)
				parseErrReport := &paintress.ExpeditionReport{
					Expedition: exp, IssueID: "?", IssueTitle: "?",
					MissionType: "?", Status: "parse_error", Reason: "report markers not found",
					FailureType: "blocker",
					PRUrl:       "none", BugIssues: "none",
				}
				if midHighCount > 0 {
					parseErrReport.HighSeverityDMails = strings.Join(midHighNames, ", ")
				}
				WriteJournal(p.config.Continent, parseErrReport)
				p.emitEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
					Expedition: exp, Status: "parse_error",
				})
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)
			case paintress.StatusSuccess:
				expSpan.AddEvent("expedition.complete",
					trace.WithAttributes(
						attribute.String("status", "success"),
						attribute.String("issue_id", report.IssueID),
						attribute.String("mission_type", report.MissionType),
					),
				)
				p.handleSuccess(report)
				p.gradient.Charge()
				p.emitEvent(domain.EventGradientChanged, domain.GradientChangedData{
					Level: p.gradient.Level(), Operator: "charge",
				})
				if matched := expedition.MidMatchedDMails(); len(matched) > 0 {
					totalTimeout := time.Duration(p.config.TimeoutSec) * time.Second
					followUpBudget := totalTimeout - time.Since(expStart)
					for _, dm := range matched {
						if dm.Action != "" {
							p.handleFeedbackAction(ctx, dm, workDir, followUpBudget)
						} else {
							p.runFollowUp(ctx, []paintress.DMail{dm}, workDir, followUpBudget)
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
				p.emitEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
					Expedition: exp, Status: "success",
					IssueID: report.IssueID, BugsFound: fmt.Sprintf("%d", report.BugsFound),
				})
				if dm := paintress.NewReportDMail(report); dm.Name != "" {
					if err := SendDMail(p.outboxStore, dm, p.eventStore); err != nil {
						p.Logger.Warn("dmail send: %v", err)
					}
				}
				for _, dm := range expedition.InboxDMails {
					if err := ArchiveInboxDMail(p.config.Continent, dm.Name, p.eventStore); err != nil {
						p.Logger.Warn("dmail archive: %v", err)
					}
				}
				p.consecutiveFailures.Store(0)
				p.totalSuccess.Add(1)
			case paintress.StatusSkipped:
				p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("issue_skipped"), report.IssueID, report.Reason))
				p.gradient.Decay()
				p.emitEvent(domain.EventGradientChanged, domain.GradientChangedData{
					Level: p.gradient.Level(), Operator: "decay",
				})
				p.writeFlag(flagDir, exp, report.IssueID, "skipped", report.Remaining, midHighCount)
				WriteJournal(p.config.Continent, report)
				p.emitEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
					Expedition: exp, Status: "skipped", IssueID: report.IssueID,
				})
				p.totalSkipped.Add(1)
			case paintress.StatusFailed:
				p.Logger.Error("%s", fmt.Sprintf(paintress.Msg("issue_failed"), report.IssueID, report.Reason))
				p.gradient.Discharge()
				p.emitEvent(domain.EventGradientChanged, domain.GradientChangedData{
					Level: p.gradient.Level(), Operator: "discharge",
				})
				p.writeFlag(flagDir, exp, report.IssueID, "failed", report.Remaining, midHighCount)
				WriteJournal(p.config.Continent, report)
				p.emitEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
					Expedition: exp, Status: "failed", IssueID: report.IssueID,
				})
				p.consecutiveFailures.Add(1)
				p.totalFailed.Add(1)
			}

			if midHighCount > 0 {
				p.Logger.Warn("Expedition #%d: %d HIGH severity D-Mail received mid-expedition", exp, midHighCount)
			}
		}

		if p.consecutiveFailures.Load() >= int64(maxConsecutiveFailures) {
			p.stageEscalation(exp, maxConsecutiveFailures)
			expSpan.AddEvent("gommage",
				trace.WithAttributes(attribute.Int("consecutive_failures", maxConsecutiveFailures)),
			)
			releaseWorkDir()
			expSpan.End()
			p.Logger.Error("%s", fmt.Sprintf(paintress.Msg("gommage"), maxConsecutiveFailures))
			p.emitEvent(domain.EventGommageTriggered, domain.GommageTriggeredData{
				Expedition: exp, ConsecutiveFailures: maxConsecutiveFailures,
			})
			return errGommage
		}

		releaseWorkDir()
		expSpan.End()
		if p.pool == nil {
			gitCmd := exec.CommandContext(ctx, "git", "checkout", p.config.BaseBranch)
			gitCmd.Dir = p.config.Continent
			_ = gitCmd.Run()
		}

		p.Logger.Info("%s", paintress.Msg("cooldown"))
		select {
		case <-time.After(10 * time.Second):
		case <-ctx.Done():
			return nil
		}
	}
}

func (p *Paintress) runReviewLoop(ctx context.Context, report *paintress.ExpeditionReport, budget time.Duration, workDir string) {
	ctx, loopSpan := platform.Tracer.Start(ctx, "review.loop",
		trace.WithAttributes(
			attribute.String("pr_url", report.PRUrl),
			attribute.String("branch", report.Branch),
		),
	)
	defer loopSpan.End()

	reviewDir := workDir
	if reviewDir == "" {
		reviewDir = p.config.Continent
	}

	var consumed time.Duration

	reviewTimeout := max(
		time.Duration(p.config.TimeoutSec)*time.Second/time.Duration(maxReviewGateCycles),
		minReviewTimeout,
	)
	var lastComments string
	for cycle := 1; cycle <= maxReviewGateCycles; cycle++ {
		if ctx.Err() != nil {
			if lastComments != "" {
				if report.Insight != "" {
					report.Insight += " | "
				}
				report.Insight += "Review interrupted: " + summarizeReview(lastComments)
			}
			p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("review_error"), ctx.Err()))
			return
		}

		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("review_running"), cycle, maxReviewGateCycles))

		_, revSpan := platform.Tracer.Start(ctx, "review.command",
			trace.WithAttributes(attribute.Int("cycle", cycle)),
		)
		reviewCtx, reviewCancel := context.WithTimeout(ctx, reviewTimeout)
		expandedCmd := ExpandReviewCmd(p.config.ReviewCmd, reviewDir, report.Branch)
		result, err := RunReview(reviewCtx, expandedCmd, reviewDir)
		reviewCancel()
		if err != nil {
			revSpan.End()
			if lastComments != "" {
				if report.Insight != "" {
					report.Insight += " | "
				}
				report.Insight += "Review interrupted: " + summarizeReview(lastComments)
			}
			p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("review_error"), err))
			return
		}

		revSpan.SetAttributes(attribute.Bool("passed", result.Passed))
		revSpan.End()

		if result.Passed {
			p.Logger.OK("%s", paintress.Msg("review_passed"))
			return
		}

		lastComments = result.Comments
		p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("review_comments"), cycle))

		branch := strings.TrimSpace(report.Branch)
		if branch == "" || strings.EqualFold(branch, "none") {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Reviewfix skipped: no valid branch"
			return
		}

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

		remaining := budget - consumed
		if remaining <= 0 {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Review not fully resolved: " + summarizeReview(lastComments)
			p.Logger.Warn("%s", paintress.Msg("review_limit"))
			return
		}

		fixCtx, fixCancel := context.WithTimeout(ctx, remaining)

		prompt := BuildReviewFixPrompt(branch, result.Comments)

		claudeCmd := p.config.ClaudeCmd
		if claudeCmd == "" {
			claudeCmd = paintress.DefaultClaudeCmd
		}

		model := p.reserve.ActiveModel()
		_, fixSpan := platform.Tracer.Start(fixCtx, "reviewfix.claude",
			trace.WithAttributes(
				attribute.Int("cycle", cycle),
				attribute.String("model", model),
			),
		)

		cmd := exec.CommandContext(fixCtx, claudeCmd,
			"--model", model,
			"--continue",
			"--allowedTools", strings.Join(ReviewFixAllowedTools, ","),
			"--dangerously-skip-permissions",
			"--print",
			"-p", prompt,
		)
		cmd.Dir = reviewDir
		cmd.WaitDelay = 3 * time.Second

		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("reviewfix_running"), model))
		start := time.Now()
		out, err := cmd.CombinedOutput()
		consumed += time.Since(start)
		fixSpan.End()
		fixCancel()

		if err != nil {
			p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("reviewfix_error"), err))
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Reviewfix failed: " + summarizeReview(string(out))
			return
		}
	}

	if report.Insight != "" {
		report.Insight += " | "
	}
	report.Insight += "Review not fully resolved: " + summarizeReview(lastComments)
	p.Logger.Warn("%s", paintress.Msg("review_limit"))
}

func (p *Paintress) runFollowUp(ctx context.Context, dmails []paintress.DMail, workDir string, remaining time.Duration) {
	if len(dmails) == 0 {
		return
	}
	if ctx.Err() != nil {
		return
	}
	if remaining <= 0 {
		p.Logger.Warn("Follow-up skipped: no remaining time budget")
		return
	}

	prompt := paintress.BuildFollowUpPrompt(dmails)
	claudeCmd := p.config.ClaudeCmd
	if claudeCmd == "" {
		claudeCmd = paintress.DefaultClaudeCmd
	}

	model := p.reserve.ActiveModel()
	_, followUpSpan := platform.Tracer.Start(ctx, "followup.claude",
		trace.WithAttributes(
			attribute.String("model", model),
			attribute.Int("matched_dmails", len(dmails)),
		),
	)
	defer followUpSpan.End()

	p.Logger.Info("Follow-up: delivering %d matched D-Mail(s) via --continue", len(dmails))

	timeout := time.Duration(p.config.TimeoutSec) * time.Second
	if remaining < timeout {
		timeout = remaining
	}
	followCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(followCtx, claudeCmd,
		"--model", model,
		"--continue",
		"--allowedTools", strings.Join(ReviewFixAllowedTools, ","),
		"--dangerously-skip-permissions",
		"--print",
		"-p", prompt,
	)
	if workDir != "" {
		cmd.Dir = workDir
	} else {
		cmd.Dir = p.config.Continent
	}
	cmd.WaitDelay = 3 * time.Second

	out, err := cmd.CombinedOutput()
	if err != nil {
		p.Logger.Warn("Follow-up failed: %v", err)
		followUpSpan.AddEvent("followup.error",
			trace.WithAttributes(attribute.String("error", err.Error())),
		)
		return
	}
	p.Logger.OK("Follow-up completed (%d bytes output)", len(out))
}

func (p *Paintress) handleSuccess(report *paintress.ExpeditionReport) {
	if report.MissionType == "verify" {
		p.Logger.Info("%s: %s", report.IssueID, report.IssueTitle)
		if report.BugsFound > 0 {
			p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("qa_bugs"), report.BugsFound, report.BugIssues))
			p.totalBugs.Add(int64(report.BugsFound))
		} else {
			p.Logger.Info("%s", paintress.Msg("qa_all_pass"))
		}
	} else {
		p.Logger.OK("%s: %s [%s]", report.IssueID, report.IssueTitle, report.MissionType)
	}
	if report.PRUrl != "" && report.PRUrl != "none" {
		p.Logger.OK("PR: %s", report.PRUrl)
	}
	if report.Remaining != "" {
		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("monolith_reads"), report.Remaining))
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

// reconcileFlags scans the continent's own flag.md and, when workers > 0,
// all worktree flag.md files, returning the one with the highest LastExpedition.
func reconcileFlags(continent string, workers int) paintress.ExpeditionFlag {
	best := ReadFlag(continent)
	if workers == 0 {
		return best
	}
	pattern := filepath.Join(continent, ".expedition", ".run", "worktrees", "*",
		".expedition", ".run", "flag.md")
	matches, _ := filepath.Glob(pattern)
	for _, match := range matches {
		base := filepath.Dir(filepath.Dir(filepath.Dir(match)))
		f := ReadFlag(base)
		if f.LastExpedition > best.LastExpedition {
			best = f
		}
	}
	return best
}

func (p *Paintress) writeFlag(dir string, expNum int, issueID, status, remaining string, midHighSeverity int) {
	current := ReadFlag(dir)
	if expNum <= current.LastExpedition {
		return
	}
	WriteFlag(dir, expNum, issueID, status, remaining, midHighSeverity)
}

func (p *Paintress) printSummary() {
	total := p.totalAttempted.Load()

	if p.config.OutputFormat == "json" {
		summary := paintress.RunSummary{
			Total:           total,
			Success:         p.totalSuccess.Load(),
			Skipped:         p.totalSkipped.Load(),
			Failed:          p.totalFailed.Load(),
			Bugs:            p.totalBugs.Load(),
			MidHighSeverity: p.totalMidHighSeverity.Load(),
			Gradient:        p.gradient.FormatLog(),
		}
		out, err := paintress.FormatSummaryJSON(summary)
		if err != nil {
			p.Logger.Error("json marshal: %v", err)
			return
		}
		fmt.Fprintln(p.DataOut, out)
		return
	}

	w := p.Logger.Writer()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "╔══════════════════════════════════════════════╗")
	fmt.Fprintln(w, "║          The Paintress rests                 ║")
	fmt.Fprintln(w, "╚══════════════════════════════════════════════╝")
	fmt.Fprintln(w)
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("expeditions_sent"), total))
	p.Logger.OK("%s", fmt.Sprintf(paintress.Msg("success_count"), p.totalSuccess.Load()))
	p.Logger.Warn("%s", fmt.Sprintf(paintress.Msg("skipped_count"), p.totalSkipped.Load()))
	p.Logger.Error("%s", fmt.Sprintf(paintress.Msg("failed_count"), p.totalFailed.Load()))
	if p.totalMidHighSeverity.Load() > 0 {
		p.Logger.Warn("Mid-expedition HIGH severity D-Mail: %d", p.totalMidHighSeverity.Load())
	}
	if p.totalBugs.Load() > 0 {
		p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("bugs_count"), p.totalBugs.Load()))
	}
	fmt.Fprintln(w)
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("gradient_info"), p.gradient.FormatLog()))
	p.Logger.Info("%s", fmt.Sprintf(paintress.Msg("party_info"), p.reserve.Status()))
	fmt.Fprintln(w)
	p.Logger.Info("Flag:     %s", paintress.FlagPath(p.config.Continent))
	p.Logger.Info("Journals: %s", paintress.JournalDir(p.config.Continent))
	p.Logger.Info("Logs:     %s", p.logDir)
}

// stageEscalation creates and stages a feedback D-Mail for escalation when
// consecutive failures reach the threshold. Errors are logged but not
// propagated — escalation is best-effort observability.
func (p *Paintress) stageEscalation(expedition, failureCount int) {
	if p.outboxStore == nil {
		return
	}
	dm := paintress.NewEscalationDMail(expedition, failureCount)
	if err := SendDMail(p.outboxStore, dm, p.eventStore); err != nil {
		p.Logger.Warn("escalation dmail: %v", err)
	}
}

// handleFeedbackAction dispatches a D-Mail based on its Action field.
// Actions: "retry" (with retry counting), "escalate", "resolve", or fallthrough.
func (p *Paintress) handleFeedbackAction(ctx context.Context, dm paintress.DMail, workDir string, remaining time.Duration) {
	switch dm.Action {
	case "retry":
		if len(dm.Issues) == 0 {
			p.Logger.Warn("Retry action without issues, falling through: %s", dm.Name)
			p.runFollowUp(ctx, []paintress.DMail{dm}, workDir, remaining)
			return
		}
		sorted := make([]string, len(dm.Issues))
		copy(sorted, dm.Issues)
		sort.Strings(sorted)
		retryKey := strings.Join(sorted, ",")

		p.retryMu.Lock()
		p.retryTracker[retryKey]++
		count := p.retryTracker[retryKey]
		p.retryMu.Unlock()

		if count > p.config.MaxRetries {
			p.Logger.Warn("Max retries (%d) reached for %s, escalating", p.config.MaxRetries, dm.Name)
			p.handleEscalation(dm)
			return
		}
		p.Logger.Info("Retry %d/%d for %s", count, p.config.MaxRetries, dm.Name)
		p.emitEvent(domain.EventRetryAttempted, map[string]any{"dmail": retryKey, "attempt": count})
		p.runFollowUp(ctx, []paintress.DMail{dm}, workDir, remaining)
	case "escalate":
		p.handleEscalation(dm)
	case "resolve":
		p.Logger.OK("Issue resolved per feedback: %s", dm.Name)
	default:
		p.runFollowUp(ctx, []paintress.DMail{dm}, workDir, remaining)
	}
}

// handleEscalation logs and emits an escalation event for a D-Mail that
// requires human attention.
func (p *Paintress) handleEscalation(dm paintress.DMail) {
	p.Logger.Warn("ESCALATION: %s requires human attention (issues: %v)", dm.Name, dm.Issues)
	p.emitEvent(domain.EventEscalated, map[string]any{"dmail": dm.Name, "issues": dm.Issues})
}
