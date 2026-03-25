package usecase

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// waveTargetProvider implements port.TargetProvider for wave-centric mode.
// Reads archive D-Mails, projects wave state, returns pending step targets.
type waveTargetProvider struct {
	archive port.ArchiveReader
}

// NewWaveTargetProvider creates a TargetProvider for wave mode.
func NewWaveTargetProvider(archive port.ArchiveReader) port.TargetProvider {
	return &waveTargetProvider{archive: archive}
}

func (p *waveTargetProvider) FetchTargets(ctx context.Context) ([]domain.ExpeditionTarget, error) {
	dmails, err := p.archive.ReadArchiveDMails(ctx)
	if err != nil {
		return nil, err
	}
	waves := domain.ProjectWaveState(dmails)
	return domain.ExpeditionTargetsFromWaves(waves), nil
}
