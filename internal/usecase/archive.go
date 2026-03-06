package usecase

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// ArchivePrune delegates to the archive ops interface.
// The ArchivePruneCommand is already valid by construction (parse-don't-validate).
func ArchivePrune(cmd domain.ArchivePruneCommand, ops port.ArchiveOps) (domain.PruneResult, error) {
	return ops.ArchivePrune(cmd.RepoPath().String(), cmd.Days().Int(), cmd.Execute())
}
