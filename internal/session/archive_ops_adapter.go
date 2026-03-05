package session

import "github.com/hironow/paintress/internal/domain"

type archiveOps struct{}

// NewArchiveOps returns a port.ArchiveOps implementation.
func NewArchiveOps() *archiveOps {
	return &archiveOps{}
}

func (*archiveOps) ArchivePrune(repoPath string, days int, execute bool) (domain.PruneResult, error) {
	return ArchivePrune(repoPath, days, execute)
}

func (*archiveOps) ListExpiredEventFiles(stateDir string, days int) ([]string, error) {
	return ListExpiredEventFiles(stateDir, days)
}

func (*archiveOps) PruneEventFiles(stateDir string, files []string) ([]string, error) {
	return PruneEventFiles(stateDir, files)
}

func (*archiveOps) PruneFlushedOutbox(repoPath string) (int, error) {
	return PruneFlushedOutbox(repoPath)
}
