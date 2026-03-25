package session

import (
	"context"

	"github.com/hironow/paintress/internal/usecase/port"
)

// inboxArchiverAdapter wraps the existing ArchiveInboxDMail function to implement port.InboxArchiver.
type inboxArchiverAdapter struct {
	emitter port.ExpeditionEventEmitter
}

// NewInboxArchiver creates an InboxArchiver adapter backed by the session-layer ArchiveInboxDMail.
func NewInboxArchiver(emitter port.ExpeditionEventEmitter) port.InboxArchiver {
	return &inboxArchiverAdapter{emitter: emitter}
}

func (a *inboxArchiverAdapter) ArchiveInboxDMail(ctx context.Context, continent, name string) error {
	return ArchiveInboxDMail(ctx, continent, name, a.emitter)
}
