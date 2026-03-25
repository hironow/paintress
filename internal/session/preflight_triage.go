package session

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
)

// triagePreFlightDMails delegates to the injected PreFlightTriager port.
// Falls back to passing all D-Mails through when no triager is set.
func (p *Paintress) triagePreFlightDMails(ctx context.Context, dmails []domain.DMail) []domain.DMail {
	if p.preFlightTriager == nil {
		return dmails
	}
	return p.preFlightTriager.TriagePreFlightDMails(ctx, dmails)
}
