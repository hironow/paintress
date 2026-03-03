package usecase

import (
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// ArchivePrune validates the ArchivePruneCommand, then delegates to session.ArchivePrune.
func ArchivePrune(cmd domain.ArchivePruneCommand) (domain.PruneResult, error) {
	if errs := cmd.Validate(); len(errs) > 0 {
		return domain.PruneResult{}, fmt.Errorf("command validation: %w", errs[0])
	}
	return session.ArchivePrune(cmd.RepoPath, cmd.Days, cmd.Execute)
}

// ListExpiredEventFiles returns .jsonl files older than the given days in the state directory.
func ListExpiredEventFiles(stateDir string, days int) ([]string, error) {
	return session.ListExpiredEventFiles(stateDir, days)
}

// PruneEventFiles deletes the named .jsonl files from the events directory.
func PruneEventFiles(stateDir string, files []string) ([]string, error) {
	return session.PruneEventFiles(stateDir, files)
}

// PruneFlushedOutbox opens the outbox DB, deletes flushed rows, runs
// incremental vacuum, and closes the store. Returns 0 if the DB does not exist.
func PruneFlushedOutbox(repoPath string) (int, error) {
	return session.PruneFlushedOutbox(repoPath)
}
