package session

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// Status collects current operational status from the event store and filesystem.
// baseDir is the repository root (the "continent" containing .expedition/).
func Status(ctx context.Context, baseDir string, logger domain.Logger) domain.StatusReport {
	report := domain.StatusReport{
		Continent: baseDir,
	}
	stateDir := filepath.Join(baseDir, domain.StateDir)
	applyLatestProviderMetadata(ctx, stateDir, &report)

	// Count inbox files
	report.InboxCount = countDirFiles(domain.InboxDir(baseDir))

	// Count archive files
	report.ArchiveCount = countDirFiles(domain.ArchiveDir(baseDir))

	// Load all events for expedition stats
	store := NewEventStore(stateDir, logger)

	allEvents, _, err := store.LoadAll()
	if err != nil {
		return report
	}
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

func applyLatestProviderMetadata(ctx context.Context, stateDir string, report *domain.StatusReport) {
	dbPath := filepath.Join(stateDir, ".run", "sessions.db")
	store, err := NewSQLiteCodingSessionStore(dbPath)
	if err != nil {
		return
	}
	defer store.Close()
	records, err := store.List(ctx, port.ListSessionOpts{Limit: 1})
	if err != nil || len(records) == 0 {
		return
	}
	meta := records[0].Metadata
	report.ProviderState = meta[domain.MetadataProviderState]
	report.ProviderReason = meta[domain.MetadataProviderReason]
	if budget := meta[domain.MetadataProviderRetryBudget]; budget != "" {
		if n, err := strconv.Atoi(budget); err == nil {
			report.ProviderRetryBudget = n
		}
	}
	if resumeAt := meta[domain.MetadataProviderResumeAt]; resumeAt != "" {
		if ts, err := time.Parse(time.RFC3339, resumeAt); err == nil {
			report.ProviderResumeAt = ts
		}
	}
	report.ProviderResumeWhen = meta[domain.MetadataProviderResumeWhen]
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
