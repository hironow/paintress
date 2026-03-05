package port

import (
	"context"
	"errors"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// ErrUnsupportedOS is returned by LocalNotifier on unsupported platforms.
var ErrUnsupportedOS = errors.New("notify: unsupported OS for local notifications")

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
	Append(events ...domain.Event) error

	// LoadAll returns all events in chronological order.
	LoadAll() ([]domain.Event, error)

	// LoadSince returns events with timestamps after the given time.
	LoadSince(after time.Time) ([]domain.Event, error)
}

// OutboxStore is the transactional outbox interface for D-Mail delivery.
// Stage writes to a write-ahead log (SQLite); Flush materialises staged
// items to archive/ and outbox/ using atomic file writes.
type OutboxStore interface {
	Stage(name string, data []byte) error
	Flush() (int, error)
	Close() error
}

// ArchiveOps handles archive pruning operations.
type ArchiveOps interface {
	ArchivePrune(repoPath string, days int, execute bool) (domain.PruneResult, error)
	ListExpiredEventFiles(stateDir string, days int) ([]string, error)
	PruneEventFiles(stateDir string, files []string) ([]string, error)
	PruneFlushedOutbox(repoPath string) (int, error)
}

// ExpeditionRunner wraps the session-layer expedition orchestrator.
type ExpeditionRunner interface {
	SetDispatcher(dispatcher EventDispatcher)
	Run(ctx context.Context) int
}

// ProjectOps handles project configuration and issue fetching.
type ProjectOps interface {
	LoadProjectConfig(absPath string) (*domain.ProjectConfig, error)
	FetchIssuesViaMCP(ctx context.Context, claudeCmd, team, project, workDir string) ([]domain.Issue, error)
}

// DoctorOps runs diagnostic checks.
type DoctorOps interface {
	RunDoctor(claudeCmd string, continent string) []domain.DoctorCheck
}
