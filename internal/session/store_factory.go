package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/eventsource"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
)

// EnsureRunDir creates the .run/ directory under stateDir if it does not exist.
// Call once before opening stores that write to .run/ (idempotent).
func EnsureRunDir(stateDir string) error {
	runDir := filepath.Join(stateDir, ".run")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("ensure run dir: %w", err)
	}
	return nil
}

// NewEventStore creates an event store for the given state directory.
// eventsource is the event persistence adapter (AWS Event Sourcing pattern).
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func NewEventStore(stateDir string, logger domain.Logger) port.EventStore {
	raw := eventsource.NewFileEventStore(eventsource.EventsDir(stateDir), logger)
	return NewSpanEventStore(raw)
}

// NewSnapshotStore creates a FileSnapshotStore at {stateDir}/snapshots/.
func NewSnapshotStore(stateDir string) port.SnapshotStore {
	return eventsource.NewFileSnapshotStore(filepath.Join(stateDir, "snapshots"))
}

// NewSeqCounter creates a SeqCounter at {stateDir}/seq.db.
// seq.db lives at stateDir root (NOT .run/) — .run/ is ephemeral
func NewSeqCounter(stateDir string) (*eventsource.SeqCounter, error) {
	return eventsource.NewSeqCounter(filepath.Join(stateDir, "seq.db"))
}

// EnsureCutover creates a SeqCounter, SnapshotStore, and raw FileEventStore,
// then runs the one-time cutover migration. Returns the SeqCounter for
// ongoing SeqNr allocation (caller must defer Close).
func EnsureCutover(ctx context.Context, stateDir, aggregateType string, logger domain.Logger) (*eventsource.SeqCounter, error) {
	sc, err := NewSeqCounter(stateDir)
	if err != nil {
		return nil, fmt.Errorf("ensure cutover: seq counter: %w", err)
	}
	ss := eventsource.NewFileSnapshotStore(filepath.Join(stateDir, "snapshots"))
	raw := eventsource.NewFileEventStore(eventsource.EventsDir(stateDir), logger)
	if _, err := eventsource.RunCutover(ctx, raw, ss, sc, aggregateType, logger); err != nil {
		sc.Close()
		return nil, fmt.Errorf("ensure cutover: %w", err)
	}
	return sc, nil
}

// NewCheckpointScanner creates a checkpoint scanner for the given continent.
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func NewCheckpointScanner(continent string) port.CheckpointScanner {
	return eventsource.NewCheckpointScanner(continent)
}

// ListExpiredEventFiles returns .jsonl files older than the given days.
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func ListExpiredEventFiles(ctx context.Context, stateDir string, days int) ([]string, error) {
	_, span := platform.Tracer.Start(ctx, "eventsource.list_expired")
	defer span.End()

	files, err := eventsource.ListExpiredEventFiles(stateDir, days)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "eventsource.list_expired"))
	}
	span.SetAttributes(attribute.Int("event.count.out", len(files)))
	return files, err
}

// PruneEventFiles deletes the named .jsonl files from the events directory.
// cmd layer should use this instead of importing eventsource directly (ADR S0008).
func PruneEventFiles(ctx context.Context, stateDir string, files []string) ([]string, error) {
	_, span := platform.Tracer.Start(ctx, "eventsource.prune")
	defer span.End()

	span.SetAttributes(attribute.Int("event.count.in", len(files)))
	deleted, err := eventsource.PruneEventFiles(stateDir, files)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "eventsource.prune"))
	}
	span.SetAttributes(attribute.Int("event.count.out", len(deleted)))
	return deleted, err
}
