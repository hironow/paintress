package session

import (
	"fmt"

	"github.com/hironow/paintress/internal/domain"
)

// WriteGommageInsight writes a gommage insight when consecutive failures trigger escalation.
// Best-effort: logs warnings on failure but does not propagate errors.
// Idempotent: InsightWriter deduplicates by title.
func WriteGommageInsight(w *InsightWriter, expedition, failureCount int) {
	entry := domain.InsightEntry{
		Title:       fmt.Sprintf("gommage-%d", expedition),
		What:        fmt.Sprintf("%d consecutive failures triggered Gommage at expedition #%d", failureCount, expedition),
		Why:         "Consecutive failures indicate systematic issue requiring intervention",
		How:         "Review failure reasons in recent journal entries",
		When:        "When consecutive failure count reaches threshold",
		Who:         fmt.Sprintf("paintress Gommage policy, expedition #%d", expedition),
		Constraints: "Counter resets on next success",
		Extra: map[string]string{
			"failure-type":  "gommage",
			"gradient-level": "0",
		},
	}

	// Best-effort: insight writing must not break main flow.
	_ = w.Append("gommage.md", "gommage", "paintress", entry)
}
