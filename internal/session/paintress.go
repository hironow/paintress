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
	"github.com/hironow/paintress/internal/harness"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
)

// sharedCircuitBreaker is the process-wide circuit breaker shared across all
// provider adapter instances. Set via SetCircuitBreaker at startup.
var sharedCircuitBreaker *platform.CircuitBreaker

// SetCircuitBreaker sets the process-wide circuit breaker for all provider calls.
// Call this once during startup before any provider invocations.
func SetCircuitBreaker(cb *platform.CircuitBreaker) {
	sharedCircuitBreaker = cb
}

// recordCircuitBreaker updates the shared circuit breaker based on provider error classification.
func recordCircuitBreaker(provider domain.Provider, err error, stderr string) {
	if sharedCircuitBreaker == nil {
		return
	}
	if err == nil {
		sharedCircuitBreaker.RecordSuccess()
		return
	}
	// Use stderr if available, otherwise try extracting from the error message itself
	classifyTarget := stderr
	if classifyTarget == "" {
		classifyTarget = err.Error()
	}
	info := harness.ClassifyProviderError(provider, classifyTarget)
	if info.IsTrip() {
		sharedCircuitBreaker.RecordProviderError(info)
	}
}

func currentProviderState() domain.ProviderStateSnapshot {
	if sharedCircuitBreaker == nil {
		return domain.ActiveProviderState()
	}
	return sharedCircuitBreaker.Snapshot()
}

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
	gradient    *harness.GradientGauge
	reserve     *harness.ReserveParty
	pool        *WorktreePool // nil when --workers=0
	notifier    port.Notifier
	approver    port.Approver
	outboxStore port.OutboxStore // transactional outbox for D-Mail delivery
	claude      port.ClaudeRunner
	seqAlloc    port.SeqAllocator

	// Retry tracking: maps sorted issue keys to attempt count
	retryTracker *harness.RetryTracker

	// PreFlightTriager processes D-Mail actions before expedition (usecase-injected)
	preFlightTriager port.PreFlightTriager

	// FeedbackActionHandler dispatches feedback D-Mail actions (usecase-injected)
	feedbackHandler port.FeedbackActionHandler

	// Wave-centric mode: target provider + tracking mode
	targetProvider port.TargetProvider
	trackingMode   domain.TrackingMode

	// Session store: owned by Paintress, closed by CloseRunner().
	sessionStore *SQLiteCodingSessionStore

	// Gommage recovery: tracks retry attempts per failure streak
	recoveryDecider   port.RecoveryDecider
	checkpointScanner port.CheckpointScanner

	// Resume targets: incomplete expeditions from previous session (workers=0 only)
	resumeTargets []IncompleteExpedition

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

func NewPaintress(cfg domain.Config, logger domain.Logger, dataOut io.Writer, errOut io.Writer, stdinIn io.Reader, emitter port.ExpeditionEventEmitter, approver port.Approver, recoveryDecider port.RecoveryDecider) *Paintress {
	if stdinIn == nil {
		stdinIn = strings.NewReader("")
	}
	if emitter == nil {
		emitter = &port.NopExpeditionEventEmitter{}
	}
	if approver == nil {
		approver = &port.AutoApprover{}
	}
	if recoveryDecider == nil {
		recoveryDecider = &port.NopRecoveryDecider{}
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
		Emitter:         emitter,
		config:          cfgCopy,
		logDir:          logDir,
		Logger:          logger,
		DataOut:         dataOut,
		ErrOut:          errOut,
		StdinIn:         stdinIn,
		gradient:        harness.NewGradientGauge(gradientMax),
		reserve:         harness.NewReserveParty(primary, reserves, logger),
		notifier:        notifier,
		approver:        approver,
		retryTracker:    harness.NewRetryTracker(),
		recoveryDecider: recoveryDecider,
	}
	runner, sessStore := NewTrackedRunner(cfgCopy, primary, logger)
	p.claude = runner
	p.sessionStore = sessStore

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

// sharedStreamBus is the process-wide session stream bus. Set via SetStreamBus
// at startup. All ClaudeAdapter instances and Expedition objects automatically
// pick up this bus.
var sharedStreamBus port.SessionStreamPublisher

// SetStreamBus sets the process-wide stream bus for live session event publishing.
// Call this once during startup before any provider invocations.
func SetStreamBus(bus port.SessionStreamPublisher) {
	sharedStreamBus = bus
}

// SharedStreamBus returns the process-wide stream bus (may be nil).
// Use this when constructing ClaudeAdapter directly outside of NewTrackedRunner.
func SharedStreamBus() port.SessionStreamPublisher {
	return sharedStreamBus
}

// NewTrackedRunner creates a ClaudeAdapter wrapped with session tracking.
// This is the standard path for resumable provider-backed invocations.
// Retry is NOT included — paintress manages retry at the expedition level.
// Store ownership: caller-owned via the returned *SQLiteCodingSessionStore.
// The caller MUST nil-check store before calling store.Close().
// Best-effort: if the session store cannot be opened, returns (adapter, nil).
func NewTrackedRunner(cfg domain.Config, model string, logger domain.Logger) (port.ClaudeRunner, *SQLiteCodingSessionStore) {
	adapter := &ClaudeAdapter{
		ClaudeCmd:  cfg.ClaudeCmd,
		Model:      model,
		TimeoutSec: cfg.TimeoutSec,
		Logger:     logger,
		StreamBus:  sharedStreamBus,
		ToolName:   "paintress",
	}
	return WrapWithSessionTracking(adapter, cfg.Continent, domain.ProviderClaudeCode, logger)
}

// WrapWithSessionTracking adds session persistence to a ClaudeRunner.
// The runner must also implement DetailedRunner for session ID capture.
// Best-effort: returns (runner, nil) when the session store cannot be opened
// or the runner does not implement DetailedRunner.
// Caller MUST nil-check store before calling store.Close().
// This is the canonical factory helper shared across all AI coding tools.
func WrapWithSessionTracking(runner port.ClaudeRunner, baseDir string, provider domain.Provider, logger domain.Logger) (port.ClaudeRunner, *SQLiteCodingSessionStore) {
	detailed, ok := runner.(port.DetailedRunner)
	if !ok {
		return runner, nil
	}
	dbPath := filepath.Join(baseDir, domain.StateDir, ".run", "sessions.db")
	store, err := NewSQLiteCodingSessionStore(dbPath)
	if err != nil {
		if logger != nil {
			logger.Debug("session tracking unavailable: %v", err)
		}
		return runner, nil
	}
	return NewSessionTrackingAdapter(detailed, store, provider), store
}

// CloseRunner closes the underlying session store opened by NewTrackedRunner.
// Store ownership: instance-owned. Caller MUST call CloseRunner when the Paintress
// instance is no longer needed (e.g. defer after NewPaintress).
// Safe to call multiple times or when no store was opened.
func (p *Paintress) CloseRunner() {
	if p.sessionStore != nil {
		p.sessionStore.Close()
		p.sessionStore = nil
	}
}

// SetEmitter sets the event emitter for the session.
// If p.seqAlloc is set, it is injected into the emitter via SetSeqAllocator.
func (p *Paintress) SetEmitter(e port.ExpeditionEventEmitter) {
	type seqSetter interface {
		SetSeqAllocator(port.SeqAllocator)
	}
	if setter, ok := e.(seqSetter); ok && p.seqAlloc != nil {
		setter.SetSeqAllocator(p.seqAlloc)
	}
	p.Emitter = e
}

// SetSeqAllocator injects a SeqAllocator for SeqNr allocation into emitted events.
func (p *Paintress) SetSeqAllocator(alloc port.SeqAllocator) {
	p.seqAlloc = alloc
}

// SetPreFlightTriager injects the pre-flight triage usecase.
func (p *Paintress) SetPreFlightTriager(t port.PreFlightTriager) {
	p.preFlightTriager = t
}

// SetTargetProvider injects the expedition target provider.
func (p *Paintress) SetTargetProvider(tp port.TargetProvider) {
	p.targetProvider = tp
}

// SetTrackingMode sets the tracking mode (wave or linear).
func (p *Paintress) SetTrackingMode(mode domain.TrackingMode) {
	p.trackingMode = mode
}

// SetCheckpointScanner injects the checkpoint scanner for resume-on-restart.
func (p *Paintress) SetCheckpointScanner(s port.CheckpointScanner) {
	p.checkpointScanner = s
}

func (p *Paintress) Run(ctx context.Context) int {
	// Acquire daemon lock — prevents multiple instances on the same continent
	runDir := filepath.Join(p.config.Continent, domain.StateDir, ".run")
	unlock, lockErr := TryLockDaemon(runDir)
	if lockErr != nil {
		p.Logger.Error("daemon lock: %v", lockErr)
		return 1
	}
	defer unlock()

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

	// Checkpoint scanning BEFORE cleanup for ALL worker modes: detect
	// incomplete expeditions so the resume/skip decision is based on
	// pre-cleanup state. Without this, cleanup removes worktrees and
	// the scanner silently skips missing dirs, losing visibility.
	allIncompletes := p.resumeIncompleteExpeditions()
	resumeWorkDirs := make(map[string]bool)
	if p.config.Workers == 0 {
		for _, inc := range allIncompletes {
			resumeWorkDirs[inc.WorkDir] = true
		}
	}

	// Best-effort: clean orphan worktrees from previous crashed sessions.
	// Exclude worktrees that are resume candidates (workers=0 only).
	p.cleanOrphanWorktrees(resumeWorkDirs)

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
			// Detached context: pool shutdown must complete even if parent ctx cancelled.
			shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
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

	// Resume policy: workers=0 resumes incomplete expeditions,
	// workers>0 starts fresh (swarm worktrees are pooled, not resumable).
	// allIncompletes was scanned BEFORE cleanup to capture full state.
	if len(allIncompletes) > 0 {
		if p.config.Workers == 0 {
			for _, inc := range allIncompletes {
				p.Logger.Info("resuming incomplete expedition #%d (phase=%s, dir=%s)", inc.Expedition, inc.Phase, inc.WorkDir)
			}
			p.resumeTargets = allIncompletes
		} else {
			p.Logger.Info("%d incomplete expedition(s) from previous session — skipping resume in swarm mode (workers=%d)", len(allIncompletes), p.config.Workers)
		}
	}

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
	WriteCorrectionInsights(
		NewInsightWriter(domain.InsightsDir(p.config.Continent), domain.RunDir(p.config.Continent)),
		preflightInbox,
		p.Logger,
	)
	if highDMails := harness.FilterHighSeverity(preflightInbox); len(highDMails) > 0 {
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
