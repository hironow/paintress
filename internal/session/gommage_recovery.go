package session

import (
	"context"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// executeRecovery performs class-specific recovery. Returns true to retry same issue.
func (p *Paintress) executeRecovery(ctx context.Context, decision domain.RecoveryDecision, exp int, _ *Expedition) bool {
	switch decision.RecoveryKind {
	case domain.RecoveryRetry:
		p.Logger.Warn("gommage recovery: %s (retry %d/%d, cooldown %s)",
			decision.Class, decision.RetryNum, decision.MaxRetry, decision.Cooldown)

		switch decision.Class {
		case domain.GommageClassTimeout:
			p.reserve.ForceReserve()
		case domain.GommageClassParseError:
			injectParseErrorLumina(p.config.Continent, p.Logger)
		}

		_ = p.Emitter.EmitGommageRecovery(exp, string(decision.Class),
			string(decision.RecoveryKind), decision.RetryNum, decision.Cooldown.String(), time.Now())

		select {
		case <-time.After(decision.Cooldown):
			return true
		case <-ctx.Done():
			return false
		}
	case domain.RecoveryHalt:
		return false
	}
	return false
}

// injectParseErrorLumina writes a corrective hint for the next expedition attempt.
func injectParseErrorLumina(continent string, logger domain.Logger) {
	w := NewInsightWriter(domain.InsightsDir(continent), domain.RunDir(continent))
	entry := domain.InsightEntry{
		Title: "parse-error-recovery",
		What:  "Previous expedition output could not be parsed",
		Why:   "Claude output did not contain expected report markers",
		How:   "Ensure output follows the exact report format with markers",
	}
	_ = w.Append("lumina-recovery.md", "recovery", "paintress", entry)
	logger.Warn("injected parse-error recovery hint into Lumina")
}

// isRateLimitError returns true if the error message indicates an API rate limit,
// not a functional failure like a merge conflict or test failure.
func isRateLimitError(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	return strings.Contains(lower, "rate_limit") ||
		strings.Contains(lower, "429") ||
		strings.Contains(lower, "quota") ||
		strings.Contains(lower, "too many requests")
}
