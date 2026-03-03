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
