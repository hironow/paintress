package session

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
)

type archiveOps struct{}

// NewArchiveOps returns a port.ArchiveOps implementation.
func NewArchiveOps() *archiveOps {
	return &archiveOps{}
}

func (*archiveOps) ArchivePrune(repoPath string, days int, execute bool) (domain.PruneResult, error) {
	return ArchivePrune(repoPath, days, execute)
}

func (*archiveOps) ListExpiredEventFiles(ctx context.Context, stateDir string, days int) ([]string, error) {
	return ListExpiredEventFiles(ctx, stateDir, days)
}

func (*archiveOps) PruneEventFiles(ctx context.Context, stateDir string, files []string) ([]string, error) {
	return PruneEventFiles(ctx, stateDir, files)
}

func (*archiveOps) PruneFlushedOutbox(repoPath string) (int, error) {
	return PruneFlushedOutbox(repoPath)
}
