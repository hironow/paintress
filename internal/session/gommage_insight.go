package session

import (
	"fmt"
	"os"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// WriteGommageInsight writes a gommage insight when consecutive failures trigger escalation.
// Best-effort: silent on failure, does not propagate errors.
// Idempotent: InsightWriter deduplicates by title.
func WriteGommageInsight(w *InsightWriter, expedition, failureCount int, continent string) {
	why := "Consecutive failures indicate systematic issue requiring intervention"
	if reasons := recentFailureReasons(continent, 5); len(reasons) > 0 {
		why = fmt.Sprintf("Recent failure reasons: %s", strings.Join(reasons, "; "))
	}

	entry := domain.InsightEntry{
		Title:       fmt.Sprintf("gommage-%d", expedition),
		What:        fmt.Sprintf("%d consecutive failures triggered Gommage at expedition #%d", failureCount, expedition),
		Why:         why,
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

// recentFailureReasons reads the last `limit` journal files and extracts
// deduplicated failure reason strings. Best-effort: returns nil on any error.
func recentFailureReasons(continent string, limit int) []string {
	files, err := ListJournalFiles(continent)
	if err != nil {
		return nil
	}

	// Take last `limit` files.
	if len(files) > limit {
		files = files[len(files)-limit:]
	}

	seen := make(map[string]struct{})
	var reasons []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "- **Reason**:") {
				reason := extractValue(line)
				if reason == "" {
					continue
				}
				if _, ok := seen[reason]; ok {
					continue
				}
				seen[reason] = struct{}{}
				reasons = append(reasons, reason)
			}
		}
	}
	return reasons
}
