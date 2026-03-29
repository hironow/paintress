package session

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// inboxReaderAdapter reads D-Mails from the inbox directory via ScanInbox.
type inboxReaderAdapter struct {
	continent string
}

// NewInboxReader creates an InboxReader for the given continent directory.
func NewInboxReader(continent string) port.InboxReader {
	return &inboxReaderAdapter{continent: continent}
}

func (a *inboxReaderAdapter) ReadInboxDMails(ctx context.Context) ([]domain.DMail, error) {
	return ScanInbox(ctx, a.continent)
}
