package usecase

import (
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// ArchivePrune validates the ArchivePruneCommand, then delegates to the archive ops interface.
func ArchivePrune(cmd domain.ArchivePruneCommand, ops port.ArchiveOps) (domain.PruneResult, error) {
	if errs := cmd.Validate(); len(errs) > 0 {
		return domain.PruneResult{}, fmt.Errorf("command validation: %w", errs[0])
	}
	return ops.ArchivePrune(cmd.RepoPath, cmd.Days, cmd.Execute)
}
