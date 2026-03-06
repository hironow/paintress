package session

import (
	"context"
	"os"
	"path/filepath"

	"go.opentelemetry.io/otel/attribute"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// Status collects current operational status from the event store and filesystem.
// baseDir is the repository root (the "continent" containing .expedition/).
func Status(ctx context.Context, baseDir string, logger domain.Logger) domain.StatusReport {
	report := domain.StatusReport{
		Continent: baseDir,
	}

	// Count inbox files
	report.InboxCount = countDirFiles(domain.InboxDir(baseDir))

	// Count archive files
	report.ArchiveCount = countDirFiles(domain.ArchiveDir(baseDir))

	// Load all events for expedition stats
	stateDir := filepath.Join(baseDir, domain.StateDir)
	store := NewEventStore(stateDir, logger)

	ctx, span := platform.Tracer.Start(ctx, "eventsource.load_all")
	defer span.End()
	allEvents, err := store.LoadAll()
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "eventsource.load_all"))
		return report
	}
	span.SetAttributes(attribute.Int("event.count.out", len(allEvents)))
	if len(allEvents) == 0 {
		return report
	}

	// Project state from events
	state := ProjectState(allEvents)
	report.Expeditions = state.TotalExpeditions
	report.Successes = state.Succeeded
	report.Failures = state.Failed
	report.GradientLevel = state.GradientLevel
	report.LastExpedition = state.LastExpeditionAt

	// Compute success rate using the domain package pure function
	report.SuccessRate = domain.SuccessRate(allEvents)

	return report
}

// countDirFiles returns the number of non-directory entries in the given directory.
// Returns 0 if the directory does not exist or cannot be read.
func countDirFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() {
			count++
		}
	}
	return count
}
