package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

const maxConsecutiveFailures = 3
const maxConsecutiveSkips = 3
const gradientMax = 5

var (
	errGommage    = errors.New("gommage: consecutive failures exceeded threshold")
	errAllSkipped = errors.New("all skipped: consecutive skips exceeded threshold")
	errComplete   = errors.New("expedition complete: no remaining issues")
)

type Paintress struct {
	Emitter     port.ExpeditionEventEmitter // event emitter wrapping aggregate + store + dispatch
	config      domain.Config
	logDir      string
	Logger      domain.Logger
	DataOut     io.Writer // stdout-equivalent for data output
	ErrOut      io.Writer // stderr-equivalent for UI chrome (banners, blank lines)
	StdinIn     io.Reader // stdin-equivalent for interactive input
	devServer   *DevServer
	gradient    *domain.GradientGauge
	reserve     *domain.ReserveParty
	pool        *WorktreePool // nil when --workers=0
	notifier    port.Notifier
	approver    port.Approver
	outboxStore port.OutboxStore // transactional outbox for D-Mail delivery
	claude      port.ClaudeRunner

	// Retry tracking: maps sorted issue keys to attempt count
	retryTracker *domain.RetryTracker

	// PreFlightTriager processes D-Mail actions before expedition (usecase-injected)
	preFlightTriager port.PreFlightTriager

	// FeedbackActionHandler dispatches feedback D-Mail actions (usecase-injected)
	feedbackHandler port.FeedbackActionHandler

	// Parallel worker same-issue guard (nil when Workers == 0)
	claimRegistry *domain.IssueClaimRegistry

	// Swarm Mode: atomic counters for concurrent worker access
	expCounter           atomic.Int64
	totalAttempted       atomic.Int64
	totalSuccess         atomic.Int64
	totalSkipped         atomic.Int64
	totalFailed          atomic.Int64
	totalBugs            atomic.Int64
	totalMidHighSeverity atomic.Int64
	consecutiveFailures  atomic.Int64
	consecutiveSkips     atomic.Int64
	escalationFired      atomic.Bool
}


func NewPaintress(cfg domain.Config, logger domain.Logger, dataOut io.Writer, errOut io.Writer, stdinIn io.Reader, emitter port.ExpeditionEventEmitter, approver port.Approver) *Paintress {
	if stdinIn == nil {
		stdinIn = strings.NewReader("")
	}
	if emitter == nil {
		emitter = &port.NopExpeditionEventEmitter{}
	}
	if approver == nil {
		approver = &port.AutoApprover{}
	}
	logDir := filepath.Join(cfg.Continent, domain.StateDir, ".run", "logs")
	os.MkdirAll(logDir, 0755)

	// Reserve Party: parse model string for reserves (already validated by ValidateProjectConfig).
	// On error, fall back to legacy single-model behavior with a warning.
	primary, reserves, parseErr := domain.ParseModelConfig(cfg.Model)
	if parseErr != nil {
		logger.Warn("ParseModelConfig: %v — falling back to raw model string %q", parseErr, cfg.Model)
		primary = cfg.Model
		reserves = nil
	}

	devDir := cfg.DevDir
	if devDir == "" {
		devDir = cfg.Continent
	}

	// Wire notifier based on config
	var notifier port.Notifier
	if cfg.NotifyCmd != "" {
		notifier = NewCmdNotifier(cfg.NotifyCmd)
	} else {
		notifier = &LocalNotifier{}
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	cfgCopy := cfg
	cfgCopy.MaxRetries = maxRetries

	p := &Paintress{
		Emitter:      emitter,
		config:       cfgCopy,
		logDir:       logDir,
		Logger:       logger,
		DataOut:      dataOut,
		ErrOut:       errOut,
		StdinIn:      stdinIn,
		gradient:     domain.NewGradientGauge(gradientMax),
		reserve:      domain.NewReserveParty(primary, reserves, logger),
		notifier:     notifier,
		approver:     approver,
		retryTracker: domain.NewRetryTracker(),
		claude: &ClaudeAdapter{
			ClaudeCmd:  cfgCopy.ClaudeCmd,
			Model:      primary,
			TimeoutSec: cfgCopy.TimeoutSec,
			Logger:     logger,
		},
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

// SetEmitter sets the event emitter for the session.
func (p *Paintress) SetEmitter(e port.ExpeditionEventEmitter) {
	p.Emitter = e
}

// SetPreFlightTriager injects the pre-flight triage usecase.
func (p *Paintress) SetPreFlightTriager(t port.PreFlightTriager) {
	p.preFlightTriager = t
}

func (p *Paintress) Run(ctx context.Context) int {
	ctx, rootSpan := platform.Tracer.Start(ctx, "paintress.run",
		trace.WithAttributes(
			attribute.String("continent", platform.SanitizeUTF8(p.config.Continent)),
			attribute.Int("max_expeditions", p.config.MaxExpeditions),
			attribute.Int("workers", p.config.Workers),
			attribute.String("model", platform.SanitizeUTF8(p.config.Model)),
			attribute.Bool("dry_run", p.config.DryRun),
		),
	)
	defer rootSpan.End()

	logPath := filepath.Join(p.logDir, fmt.Sprintf("paintress-%s.log", time.Now().Format("20060102")))
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		p.Logger.Warn("log file: %v", err)
	} else {
		type extraWriterSetter interface {
			SetExtraWriter(w io.Writer)
		}
		if setter, ok := p.Logger.(extraWriterSetter); ok {
			setter.SetExtraWriter(logFile)
		}
		defer logFile.Close()
	}

	// Initialize transactional outbox store for D-Mail delivery.
	outboxStore, err := NewOutboxStoreForDir(p.config.Continent)
	if err != nil {
		p.Logger.Error("outbox store: %v", err)
		rootSpan.End()
		return 1
	}
	defer outboxStore.Close()
	p.outboxStore = outboxStore

	p.printBanner()
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("continent"), p.config.Continent))

	// Start dev server (stays alive across expeditions)
	if !p.config.DryRun && p.devServer != nil {
		if err := p.devServer.Start(ctx); err != nil {
			p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("devserver_warn"), err))
		}
		defer p.devServer.Stop()
	}

	// Initialize worktree pool and claim registry if workers > 0.
	if p.config.Workers > 0 {
		p.claimRegistry = domain.NewIssueClaimRegistry()
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
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("monolith_reads"), monolith.Remaining))
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("max_expeditions"), p.config.MaxExpeditions))
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("party_info"), p.reserve.Status()))
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("gradient_info"), p.gradient.FormatForPrompt()))
	p.Logger.Info("%s", fmt.Sprintf(domain.Msg("timeout_info"), p.config.TimeoutSec))
	claudeCmd := p.config.ClaudeCmd
	if claudeCmd != domain.DefaultClaudeCmd {
		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("claude_cmd_info"), claudeCmd))
	}
	if p.config.DryRun {
		p.Logger.Warn("%s", domain.Msg("dry_run"))
	}
	fmt.Fprintln(p.ErrOut)

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
		p.Logger.OK("%s", fmt.Sprintf(domain.Msg("lumina_extracted"), len(luminas)))

		// Best-effort: write Lumina insights for cross-tool observability
		insightWriter := NewInsightWriter(
			domain.InsightsDir(p.config.Continent),
			domain.RunDir(p.config.Continent),
		)
		WriteLuminaInsights(insightWriter, luminas)
	}

	// Pre-flight HIGH severity gate (once, before workers start).
	preflightInbox, scanErr := ScanInbox(ctx, p.config.Continent)
	if scanErr != nil {
		p.Logger.Error("inbox scan failed (fail-closed): %v", scanErr)
		return 1
	}
	if highDMails := domain.FilterHighSeverity(preflightInbox); len(highDMails) > 0 {
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

	fmt.Fprintln(p.ErrOut)
	p.printSummary()

	switch {
	case errors.Is(err, errComplete):
		return 0
	case errors.Is(err, errGommage):
		return 1
	case errors.Is(err, errAllSkipped):
		return 1
	case ctx.Err() != nil:
		return 130
	case err != nil:
		return 1
	default:
		return 0
	}
}
