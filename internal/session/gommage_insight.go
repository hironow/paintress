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
func WriteGommageInsight(w *InsightWriter, expedition, failureCount int, continent string, class domain.GommageClass) {
	why := "Consecutive failures indicate systematic issue requiring intervention"
	if reasons := recentFailureReasons(continent, 5); len(reasons) > 0 {
		// Deduplicate for human-readable insight output (raw reasons used by classifier)
		why = fmt.Sprintf("Recent failure reasons: %s", strings.Join(dedupStrings(reasons), "; "))
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
			"failure-type":   "gommage",
			"gradient-level": "0",
			"gommage-class":  string(class),
		},
	}

	// Best-effort: insight writing must not break main flow.
	_ = w.Append("gommage.md", "gommage", "paintress", entry)
}

// recentFailureReasons reads the last `limit` journal files and extracts
// raw (non-deduplicated) failure reason strings from the last N journals.
// Deduplication was removed because ClassifyGommage needs frequency counts
// to determine the majority class (e.g. 3× "timeout" out of 5 reasons).
// Best-effort: returns nil on any error.
func recentFailureReasons(continent string, limit int) []string {
	files, err := ListJournalFiles(continent)
	if err != nil {
		return nil
	}

	// Take last `limit` files.
	if len(files) > limit {
		files = files[len(files)-limit:]
	}

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
				reasons = append(reasons, reason)
			}
		}
	}
	return reasons
}

// dedupStrings returns unique strings preserving first-seen order.
func dedupStrings(ss []string) []string {
	seen := make(map[string]struct{}, len(ss))
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
