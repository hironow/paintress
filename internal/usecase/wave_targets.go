package usecase

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// waveTargetProvider implements port.TargetProvider for wave-centric mode.
// Reads archive + inbox D-Mails, projects wave state, returns pending step targets.
type waveTargetProvider struct {
	archive port.ArchiveReader
	inbox   port.InboxReader
}

// NewWaveTargetProvider creates a TargetProvider for wave mode.
// Both archive and inbox are read: archive contains completion reports,
// inbox contains specs not yet archived (first expedition cycle).
func NewWaveTargetProvider(archive port.ArchiveReader, inbox port.InboxReader) port.TargetProvider {
	return &waveTargetProvider{archive: archive, inbox: inbox}
}

func (p *waveTargetProvider) FetchTargets(ctx context.Context) ([]domain.ExpeditionTarget, error) {
	archived, err := p.archive.ReadArchiveDMails(ctx)
	if err != nil {
		return nil, err
	}
	// Inbox read failure is non-fatal: fall back to archive-only.
	// A malformed file in inbox must not block target resolution.
	inboxDMails, _ := p.inbox.ReadInboxDMails(ctx)
	// Archive first (has completion reports), then inbox (new specs).
	// ProjectWaveState uses first-spec-wins, so archive spec takes precedence.
	dmails := append(archived, inboxDMails...)
	waves := domain.ProjectWaveState(dmails)
	return domain.ExpeditionTargetsFromWaves(waves), nil
}
