// nosemgrep: structure.multiple-exported-interfaces-go,structure.multiple-exported-structs-go,structure.exported-struct-and-interface-go -- port interface family (all interfaces are usecase-layer ports for dependency inversion; Nop implementations are null objects sealed with their interface); splitting each port to its own file would scatter the contract surface with no cohesion benefit [permanent]
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
type CheckpointScanner interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster; all interfaces in this file are usecase/port contracts (CheckpointScanner/RecoveryDecider/InitRunner/EventDispatcher/Approver/Notifier/GitExecutor/PolicyMetrics/ContextEventApplier/EventStore/SnapshotStore/SeqAllocator/OutboxStore/ArchiveOps/ArchiveReader/InboxReader/StepProgressReader/TargetProvider/PreFlightTriager/FeedbackActionHandler/FollowUpRunner/InboxArchiver/ExpeditionEventEmitter/ExpeditionRunner/ProjectOps/DoctorOps/RunLockStore); splitting would fragment the port contract file that cmd uses as composition root [permanent]
	// FindIncompleteCheckpoints returns checkpoint events that have no
	// subsequent expedition.completed event for the same expedition number.
	FindIncompleteCheckpoints() []domain.ExpeditionCheckpointData
}

// NopCheckpointScanner returns no incomplete checkpoints.
type NopCheckpointScanner struct{} // nosemgrep: structure.exported-struct-and-interface-go,structure.multiple-exported-structs-go -- null-object for CheckpointScanner; must co-locate with interface definition; port null-object family cohesive set [permanent]

func (*NopCheckpointScanner) FindIncompleteCheckpoints() []domain.ExpeditionCheckpointData {
	return nil
}

// RecoveryDecider classifies failure streaks and decides retry vs halt.
// Implemented by domain.ExpeditionAggregate, injected into session.
type RecoveryDecider interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	DecideRecovery(reasons []string) domain.RecoveryDecision
	ResetRecovery()
}

// NopRecoveryDecider always halts (no recovery). Used as default when no aggregate is provided.
type NopRecoveryDecider struct{} // nosemgrep: structure.exported-struct-and-interface-go,structure.multiple-exported-structs-go -- null-object for RecoveryDecider; must co-locate with interface definition; port null-object family cohesive set [permanent]

func (*NopRecoveryDecider) DecideRecovery(_ []string) domain.RecoveryDecision {
	return domain.RecoveryDecision{RecoveryKind: domain.RecoveryHalt, Class: domain.GommageClassSystematic}
}
func (*NopRecoveryDecider) ResetRecovery() {}

// InitOption configures optional behavior for project initialization.
type InitOption func(*InitConfig)

// InitConfig holds per-invocation configuration for project initialization.
// Tools use only the fields relevant to their init flow.
type InitConfig struct { // nosemgrep: structure.exported-struct-and-interface-go,structure.multiple-exported-structs-go -- port DTO/null-object family: InitConfig co-locates with InitRunner; Nop* types are null-objects sealed with their interfaces; all required by cmd composition root [permanent]
	Team       string
	Project    string
	Lang       string
	Strictness string
}

// ApplyInitOptions applies InitOption functions to an InitConfig and returns it.
func ApplyInitOptions(opts ...InitOption) InitConfig {
	var c InitConfig
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// WithTeam sets the team identifier for project initialization.
func WithTeam(t string) InitOption { return func(c *InitConfig) { c.Team = t } }

// WithProject sets the project name for initialization.
func WithProject(p string) InitOption { return func(c *InitConfig) { c.Project = p } }

// WithLang sets the language for initialization (e.g. "ja", "en").
func WithLang(l string) InitOption { return func(c *InitConfig) { c.Lang = l } }

// WithStrictness sets the strictness level (e.g. "fog", "alert", "lockdown").
func WithStrictness(s string) InitOption { return func(c *InitConfig) { c.Strictness = s } }

// InitRunner handles project initialization I/O.
// Returns warnings for non-fatal issues (nil when none). Error for critical failures.
type InitRunner interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	InitProject(baseDir string, opts ...InitOption) (warnings []string, err error)
}

// EventDispatcher dispatches domain events to policy handlers.
// Implemented by usecase.PolicyEngine; injected into session via struct field.
type EventDispatcher interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	Dispatch(ctx context.Context, event domain.Event) error
}

// Approver requests human approval and blocks until a response or context expiry.
type Approver interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	RequestApproval(ctx context.Context, message string) (approved bool, err error)
}

// AutoApprover always approves without human interaction.
type AutoApprover struct{} // nosemgrep: structure.exported-struct-and-interface-go,structure.multiple-exported-structs-go -- null-object for Approver; must co-locate with interface definition; port null-object family cohesive set [permanent]

func (*AutoApprover) RequestApproval(_ context.Context, _ string) (bool, error) {
	return true, nil
}

// Notifier sends fire-and-forget notifications to the human operator.
type Notifier interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	Notify(ctx context.Context, title, message string) error
}

// NopNotifier is a no-op notifier for quiet mode or testing.
type NopNotifier struct{} // nosemgrep: structure.exported-struct-and-interface-go,structure.multiple-exported-structs-go -- null-object for Notifier; must co-locate with interface definition; port null-object family cohesive set [permanent]

func (*NopNotifier) Notify(_ context.Context, _, _ string) error {
	return nil
}

// GitExecutor abstracts git command execution for testability.
type GitExecutor interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	Git(ctx context.Context, dir string, args ...string) ([]byte, error)
	Shell(ctx context.Context, dir string, command string) ([]byte, error)
}

// PolicyMetrics records policy handler execution metrics.
type PolicyMetrics interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	RecordPolicyEvent(ctx context.Context, eventType string, status string)
}

// NopPolicyMetrics is a no-op metrics recorder for tests and quiet mode.
type NopPolicyMetrics struct{} // nosemgrep: structure.exported-struct-and-interface-go,structure.multiple-exported-structs-go -- null-object for PolicyMetrics; must co-locate with interface definition; port null-object family cohesive set [permanent]

func (*NopPolicyMetrics) RecordPolicyEvent(_ context.Context, _, _ string) {}

// ContextEventApplier extends domain.EventApplier with context propagation.
// domain.EventApplier is ctx-free (pure domain); this port interface adds ctx
// so that session-layer implementations can propagate trace/cancel.
type ContextEventApplier interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	Apply(ctx context.Context, event domain.Event) error
	Rebuild(ctx context.Context, events []domain.Event) error
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
}

// EventStore is the append-only event persistence interface.
type EventStore interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	// Append persists one or more events. Validation is performed before any writes.
	Append(ctx context.Context, events ...domain.Event) (domain.AppendResult, error)

	// LoadAll returns all events in chronological order.
	LoadAll(ctx context.Context) ([]domain.Event, domain.LoadResult, error)

	// LoadSince returns events with timestamps after the given time.
	LoadSince(ctx context.Context, after time.Time) ([]domain.Event, domain.LoadResult, error)

	// LoadAfterSeqNr returns all events with SeqNr > afterSeqNr,
	// ordered by SeqNr ascending. Used for snapshot-based recovery.
	LoadAfterSeqNr(ctx context.Context, afterSeqNr uint64) ([]domain.Event, domain.LoadResult, error)

	// LatestSeqNr returns the highest recorded SeqNr across all events.
	// Returns 0 if no events have a SeqNr assigned.
	LatestSeqNr(ctx context.Context) (uint64, error)
}

// SnapshotStore persists materialized projection state at a known SeqNr.
// Snapshots are an optimization — the system must function without them
// (falling back to full replay via LoadAll).
type SnapshotStore interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	// Save persists a snapshot. aggregateType identifies the projection kind.
	Save(ctx context.Context, aggregateType string, seqNr uint64, state []byte) error

	// Load returns the latest snapshot for the given aggregateType.
	// Returns (0, nil, nil) if no snapshot exists.
	Load(ctx context.Context, aggregateType string) (seqNr uint64, state []byte, err error)
}

// SeqAllocator assigns globally monotonic sequence numbers to events.
// Implemented by eventsource.SeqCounter (SQLite-backed).
type SeqAllocator interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	AllocSeqNr(ctx context.Context) (uint64, error)
}

// OutboxStore is the transactional outbox interface for D-Mail delivery.
// Stage writes to a write-ahead log (SQLite); Flush materialises staged
// items to archive/ and outbox/ using atomic file writes.
type OutboxStore interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	Stage(ctx context.Context, name string, data []byte) error
	Flush(ctx context.Context) (int, error)
	Close() error
}

// ArchiveOps handles archive pruning operations.
type ArchiveOps interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	ArchivePrune(repoPath string, days int, execute bool) (domain.PruneResult, error)
	ListExpiredEventFiles(ctx context.Context, stateDir string, days int) ([]string, error)
	PruneEventFiles(ctx context.Context, stateDir string, files []string) ([]string, error)
	PruneFlushedOutbox(ctx context.Context, repoPath string) (int, error)
}

// ArchiveReader reads D-Mails from the archive directory.
// Implemented in session layer (filesystem I/O), injected by cmd.
type ArchiveReader interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	ReadArchiveDMails(ctx context.Context) ([]domain.DMail, error)
}

// InboxReader reads D-Mails from the inbox directory.
// Implemented in session layer (filesystem I/O), injected by cmd.
type InboxReader interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	ReadInboxDMails(ctx context.Context) ([]domain.DMail, error)
}

// StepProgressReader reads the WaveStepProgress Read Model from the event store.
// Replays spec.registered and expedition.completed events to build step completion state.
type StepProgressReader interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	ReadStepProgress(ctx context.Context) (*domain.WaveStepProgress, error)
}

// TargetProvider supplies expedition targets based on tracking mode.
// Wave mode: reads archive + inbox, projects wave state, returns pending steps.
// Linear mode: fetches issues from Linear MCP.
type TargetProvider interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	FetchTargets(ctx context.Context) ([]domain.ExpeditionTarget, error)
}

// PreFlightTriager processes inbox D-Mails before expedition creation.
// Implemented in usecase layer, injected into session by cmd (composition root).
type PreFlightTriager interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	TriagePreFlightDMails(ctx context.Context, dmails []domain.DMail) []domain.DMail
}

// FeedbackActionHandler processes feedback D-Mail actions.
// Implemented in usecase layer, injected into session by cmd (composition root).
type FeedbackActionHandler interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	HandleFeedbackAction(ctx context.Context, dm domain.DMail, workDir string, remaining time.Duration)
}

// FollowUpRunner executes follow-up expeditions triggered by D-Mail feedback.
// Implemented in session layer (Claude CLI subprocess), injected into usecase by cmd.
type FollowUpRunner interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	RunFollowUp(ctx context.Context, dmails []domain.DMail, workDir string, remaining time.Duration)
}

// InboxArchiver archives consumed inbox D-Mails to the archive directory.
// Implemented in session layer (filesystem I/O), injected into usecase by cmd.
type InboxArchiver interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	ArchiveInboxDMail(ctx context.Context, continent, name string) error
}

// ExpeditionEventEmitter wraps aggregate event production + persistence + dispatch
// for expedition operations. Implemented in usecase layer, injected into session.
// Dispatch is best-effort: errors are logged but not returned.
type ExpeditionEventEmitter interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	EmitStartExpedition(expedition, worker int, model string, now time.Time) error
	EmitCompleteExpedition(expedition int, status, issueID, bugsFound, waveID, stepID string, now time.Time) error
	EmitSpecRegistered(waveID string, steps []domain.WaveStepDef, source string, now time.Time) error
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
type NopExpeditionEventEmitter struct{} // nosemgrep: structure.exported-struct-and-interface-go,structure.multiple-exported-structs-go -- null-object for ExpeditionEventEmitter; must co-locate with interface definition; port null-object family cohesive set [permanent]

func (*NopExpeditionEventEmitter) EmitStartExpedition(_, _ int, _ string, _ time.Time) error {
	return nil
}
func (*NopExpeditionEventEmitter) EmitCompleteExpedition(_ int, _, _, _, _, _ string, _ time.Time) error { // nosemgrep: domain-primitives.multiple-string-params-go -- Nop implementation of EmitCompleteExpedition interface [permanent]
	return nil
}
func (*NopExpeditionEventEmitter) EmitSpecRegistered(_ string, _ []domain.WaveStepDef, _ string, _ time.Time) error {
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
type ExpeditionRunner interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	SetEmitter(emitter ExpeditionEventEmitter)
	SetPreFlightTriager(triager PreFlightTriager)
	SetFeedbackHandler(handler FeedbackActionHandler)
	SetTargetProvider(tp TargetProvider)
	SetTrackingMode(mode domain.TrackingMode)
	Run(ctx context.Context) int
}

// ProjectOps handles project configuration and issue fetching.
type ProjectOps interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	LoadProjectConfig(absPath string) (*domain.ProjectConfig, error)
	FetchIssuesViaMCP(ctx context.Context, runner ProviderRunner, team, project, workDir string) ([]domain.Issue, error)
}

// DoctorOps runs diagnostic checks.
type DoctorOps interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	RunDoctor(ctx context.Context, claudeCmd string, continent string, repair bool, mode domain.TrackingMode) []domain.DoctorCheck
}

// RunLockStore provides cross-process run locking backed by persistent storage.
// Prevents duplicate runs when multiple CLI instances target the same state directory.
type RunLockStore interface { // nosemgrep: structure.multiple-exported-interfaces-go -- port interface cluster cohesive set; see CheckpointScanner [permanent]
	// TryAcquire attempts to acquire a lock for the given run key.
	// Returns (true, "", nil) if acquired, (false, holder, nil) if already held.
	// Stale locks (past expires_at) are automatically cleaned up.
	TryAcquire(ctx context.Context, runKey string, ttl time.Duration) (acquired bool, holder string, err error)
	// Release releases a lock previously acquired by this holder.
	Release(ctx context.Context, runKey string, holder string) error
	// IsHeld returns whether the lock is currently held and by whom.
	IsHeld(ctx context.Context, runKey string) (held bool, holder string, err error)
	// Close releases database resources.
	Close() error
}
