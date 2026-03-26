package port

import (
	"context"
	"errors"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// ErrUnsupportedOS is returned by LocalNotifier on unsupported platforms.
var ErrUnsupportedOS = errors.New("notify: unsupported OS for local notifications")

// CheckpointScanner finds incomplete expeditions from the event store.
// Implemented in eventsource layer, injected into session.
type CheckpointScanner interface {
	// FindIncompleteCheckpoints returns checkpoint events that have no
	// subsequent expedition.completed event for the same expedition number.
	FindIncompleteCheckpoints() []domain.ExpeditionCheckpointData
}

// NopCheckpointScanner returns no incomplete checkpoints.
type NopCheckpointScanner struct{}

func (*NopCheckpointScanner) FindIncompleteCheckpoints() []domain.ExpeditionCheckpointData {
	return nil
}

// RecoveryDecider classifies failure streaks and decides retry vs halt.
// Implemented by domain.ExpeditionAggregate, injected into session.
type RecoveryDecider interface {
	DecideRecovery(reasons []string) domain.RecoveryDecision
	ResetRecovery()
}

// NopRecoveryDecider always halts (no recovery). Used as default when no aggregate is provided.
type NopRecoveryDecider struct{}

func (*NopRecoveryDecider) DecideRecovery(_ []string) domain.RecoveryDecision {
	return domain.RecoveryDecision{RecoveryKind: domain.RecoveryHalt, Class: domain.GommageClassSystematic}
}
func (*NopRecoveryDecider) ResetRecovery() {}

// InitRunner handles project initialization I/O.
type InitRunner interface {
	InitProject(repoPath, team, project string) error
}

// EventDispatcher dispatches domain events to policy handlers.
// Implemented by usecase.PolicyEngine; injected into session via struct field.
type EventDispatcher interface {
	Dispatch(ctx context.Context, event domain.Event) error
}

// Approver requests human approval and blocks until a response or context expiry.
type Approver interface {
	RequestApproval(ctx context.Context, message string) (approved bool, err error)
}

// AutoApprover always approves without human interaction.
type AutoApprover struct{}

func (*AutoApprover) RequestApproval(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// Notifier sends fire-and-forget notifications to the human operator.
type Notifier interface {
	Notify(ctx context.Context, title, message string) error
}

// NopNotifier is a no-op notifier for quiet mode or testing.
type NopNotifier struct{}

func (*NopNotifier) Notify(_ context.Context, _, _ string) error {
	return nil
}

// GitExecutor abstracts git command execution for testability.
type GitExecutor interface {
	Git(ctx context.Context, dir string, args ...string) ([]byte, error)
	Shell(ctx context.Context, dir string, command string) ([]byte, error)
}

// PolicyMetrics records policy handler execution metrics.
type PolicyMetrics interface {
	RecordPolicyEvent(ctx context.Context, eventType string, status string)
}

// NopPolicyMetrics is a no-op metrics recorder for tests and quiet mode.
type NopPolicyMetrics struct{}

func (*NopPolicyMetrics) RecordPolicyEvent(_ context.Context, _, _ string) {}

// EventStore is the append-only event persistence interface.
type EventStore interface {
	// Append persists one or more events. Validation is performed before any writes.
	Append(events ...domain.Event) (domain.AppendResult, error)

	// LoadAll returns all events in chronological order.
	LoadAll() ([]domain.Event, domain.LoadResult, error)

	// LoadSince returns events with timestamps after the given time.
	LoadSince(after time.Time) ([]domain.Event, domain.LoadResult, error)
}

// OutboxStore is the transactional outbox interface for D-Mail delivery.
// Stage writes to a write-ahead log (SQLite); Flush materialises staged
// items to archive/ and outbox/ using atomic file writes.
type OutboxStore interface {
	Stage(ctx context.Context, name string, data []byte) error
	Flush(ctx context.Context) (int, error)
	Close() error
}

// ArchiveOps handles archive pruning operations.
type ArchiveOps interface {
	ArchivePrune(repoPath string, days int, execute bool) (domain.PruneResult, error)
	ListExpiredEventFiles(ctx context.Context, stateDir string, days int) ([]string, error)
	PruneEventFiles(ctx context.Context, stateDir string, files []string) ([]string, error)
	PruneFlushedOutbox(ctx context.Context, repoPath string) (int, error)
}

// ArchiveReader reads D-Mails from the archive directory.
// Implemented in session layer (filesystem I/O), injected by cmd.
type ArchiveReader interface {
	ReadArchiveDMails(ctx context.Context) ([]domain.DMail, error)
}

// TargetProvider supplies expedition targets based on tracking mode.
// Wave mode: reads archive, projects wave state, returns pending steps.
// Linear mode: fetches issues from Linear MCP.
type TargetProvider interface {
	FetchTargets(ctx context.Context) ([]domain.ExpeditionTarget, error)
}

// PreFlightTriager processes inbox D-Mails before expedition creation.
// Implemented in usecase layer, injected into session by cmd (composition root).
type PreFlightTriager interface {
	TriagePreFlightDMails(ctx context.Context, dmails []domain.DMail) []domain.DMail
}

// FeedbackActionHandler processes feedback D-Mail actions.
// Implemented in usecase layer, injected into session by cmd (composition root).
type FeedbackActionHandler interface {
	HandleFeedbackAction(ctx context.Context, dm domain.DMail, workDir string, remaining time.Duration)
}

// FollowUpRunner executes follow-up expeditions triggered by D-Mail feedback.
// Implemented in session layer (Claude CLI subprocess), injected into usecase by cmd.
type FollowUpRunner interface {
	RunFollowUp(ctx context.Context, dmails []domain.DMail, workDir string, remaining time.Duration)
}

// InboxArchiver archives consumed inbox D-Mails to the archive directory.
// Implemented in session layer (filesystem I/O), injected into usecase by cmd.
type InboxArchiver interface {
	ArchiveInboxDMail(ctx context.Context, continent, name string) error
}

// ExpeditionEventEmitter wraps aggregate event production + persistence + dispatch
// for expedition operations. Implemented in usecase layer, injected into session.
// Dispatch is best-effort: errors are logged but not returned.
type ExpeditionEventEmitter interface {
	EmitStartExpedition(expedition, worker int, model string, now time.Time) error
	EmitCompleteExpedition(expedition int, status, issueID, bugsFound string, now time.Time) error
	EmitInboxReceived(name, severity string, now time.Time) error
	EmitGommage(expedition int, now time.Time) error
	EmitGradientChange(level int, operator string, now time.Time) error
	EmitRetryAttempted(dmailKey string, attempt int, now time.Time) error
	EmitEscalated(dmailName string, issues []string, now time.Time) error
	EmitResolved(dmailName string, issues []string, now time.Time) error
	EmitDMailStaged(name string, now time.Time) error
	EmitDMailFlushed(count int, now time.Time) error
	EmitDMailArchived(name string, now time.Time) error
	EmitGommageRecovery(expedition int, class, action string, retryNum int, cooldown string, now time.Time) error
	EmitCheckpoint(expedition int, phase, workDir string, commitCount int, now time.Time) error
}

// NopExpeditionEventEmitter is a no-op emitter for tests and when event
// sourcing is not configured. All methods return nil.
type NopExpeditionEventEmitter struct{}

func (*NopExpeditionEventEmitter) EmitStartExpedition(_, _ int, _ string, _ time.Time) error {
	return nil
}
func (*NopExpeditionEventEmitter) EmitCompleteExpedition(_ int, _, _, _ string, _ time.Time) error {
	return nil
}
func (*NopExpeditionEventEmitter) EmitInboxReceived(_, _ string, _ time.Time) error { return nil }
func (*NopExpeditionEventEmitter) EmitGommage(_ int, _ time.Time) error             { return nil }
func (*NopExpeditionEventEmitter) EmitGradientChange(_ int, _ string, _ time.Time) error {
	return nil
}
func (*NopExpeditionEventEmitter) EmitRetryAttempted(_ string, _ int, _ time.Time) error { return nil }
func (*NopExpeditionEventEmitter) EmitEscalated(_ string, _ []string, _ time.Time) error { return nil }
func (*NopExpeditionEventEmitter) EmitResolved(_ string, _ []string, _ time.Time) error  { return nil }
func (*NopExpeditionEventEmitter) EmitDMailStaged(_ string, _ time.Time) error           { return nil }
func (*NopExpeditionEventEmitter) EmitDMailFlushed(_ int, _ time.Time) error             { return nil }
func (*NopExpeditionEventEmitter) EmitDMailArchived(_ string, _ time.Time) error         { return nil }
func (*NopExpeditionEventEmitter) EmitGommageRecovery(_ int, _, _ string, _ int, _ string, _ time.Time) error {
	return nil
}
func (*NopExpeditionEventEmitter) EmitCheckpoint(_ int, _, _ string, _ int, _ time.Time) error {
	return nil
}

// ExpeditionRunner wraps the session-layer expedition orchestrator.
type ExpeditionRunner interface {
	SetEmitter(emitter ExpeditionEventEmitter)
	SetPreFlightTriager(triager PreFlightTriager)
	SetFeedbackHandler(handler FeedbackActionHandler)
	SetTargetProvider(tp TargetProvider)
	SetTrackingMode(mode domain.TrackingMode)
	Run(ctx context.Context) int
}

// ProjectOps handles project configuration and issue fetching.
type ProjectOps interface {
	LoadProjectConfig(absPath string) (*domain.ProjectConfig, error)
	FetchIssuesViaMCP(ctx context.Context, runner ClaudeRunner, team, project, workDir string) ([]domain.Issue, error)
}

// DoctorOps runs diagnostic checks.
type DoctorOps interface {
	RunDoctor(claudeCmd string, continent string, repair bool) []domain.DoctorCheck
}
