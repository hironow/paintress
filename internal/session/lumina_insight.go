package session

import (
	"fmt"

	"github.com/hironow/paintress/internal/domain"
)

// WriteLuminaInsights writes insight entries for each discovered Lumina pattern.
// Best-effort: silent on failure, does not propagate errors.
// Idempotent: InsightWriter deduplicates by title.
func WriteLuminaInsights(w *InsightWriter, luminas []domain.Lumina) {
	if len(luminas) == 0 {
		return
	}

	for _, l := range luminas {
		entry := luminaToInsight(l)
		if err := w.Append("lumina.md", "lumina", "paintress", entry); err != nil {
			// Best-effort: insight writing must not break main flow.
			continue
		}
	}
}

// luminaToInsight maps a Lumina to an InsightEntry with the 6 required axes.
func luminaToInsight(l domain.Lumina) domain.InsightEntry {
	entry := domain.InsightEntry{
		Title:       l.Pattern,
		What:        l.Pattern,
		Constraints: fmt.Sprintf("appeared %d times", l.Uses),
		Extra:       map[string]string{"failure-type": l.Source},
	}

	switch l.Source {
	case "failure-pattern":
		entry.Why = "Repeated failure pattern detected in expedition journals"
		entry.How = "Apply defensive strategy: avoid this failure mode"
		entry.When = "Before each expedition attempt"
		entry.Who = "paintress Lumina scanner"
	case "success-pattern":
		entry.Why = "Proven successful pattern across multiple expeditions"
		entry.How = "Continue doing: reinforce this approach"
		entry.When = "During expedition planning"
		entry.Who = "paintress Lumina scanner"
	case "high-severity-alert":
		entry.Why = "HIGH severity D-Mail received in past expedition"
		entry.How = "Requires immediate attention before proceeding"
		entry.When = "Pre-flight check before expedition"
		entry.Who = "paintress Lumina scanner"
	default:
		entry.Why = "Pattern detected in expedition journals"
		entry.How = "Review and incorporate into expedition strategy"
		entry.When = "During expedition planning"
		entry.Who = "paintress Lumina scanner"
	}

	return entry
}
