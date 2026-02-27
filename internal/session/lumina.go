package session

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/alitto/pond/v2"
	"github.com/hironow/paintress"
)

// ScanJournalsForLumina reads all journal files in parallel goroutines,
// extracts failure reasons and success patterns, and returns Luminas.
func ScanJournalsForLumina(continent string) []paintress.Lumina {
	files, err := ListJournalFiles(continent)
	if err != nil || len(files) == 0 {
		return nil
	}

	type journalData struct {
		status       string
		reason       string
		mission      string
		issue        string
		insight      string
		highSeverity string
	}

	// Parallel journal scanning with bounded concurrency
	pool := pond.NewResultPool[journalData](runtime.GOMAXPROCS(0))
	defer pool.StopAndWait()
	group := pool.NewGroup()

	for _, f := range files {
		group.Submit(func() journalData {
			content, err := os.ReadFile(f)
			if err != nil {
				return journalData{}
			}
			text := string(content)

			entry := journalData{}
			for _, line := range strings.Split(text, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "- **Status**:") {
					entry.status = extractValue(line)
				} else if strings.HasPrefix(line, "- **Reason**:") {
					entry.reason = extractValue(line)
				} else if strings.HasPrefix(line, "- **Mission**:") {
					entry.mission = extractValue(line)
				} else if strings.HasPrefix(line, "- **Issue**:") {
					entry.issue = extractValue(line)
				} else if strings.HasPrefix(line, "- **Insight**:") {
					entry.insight = extractValue(line)
				} else if strings.HasPrefix(line, "- **HIGH severity D-Mail**:") {
					entry.highSeverity = extractValue(line)
				}
			}

			return entry
		})
	}

	entries, err := group.Wait()
	if err != nil {
		return nil
	}

	// Aggregate patterns
	failureReasons := make(map[string]int)
	successPatterns := make(map[string]int)
	highSeverityAlerts := make(map[string]int)

	for _, e := range entries {
		if e.status == "failed" {
			key := e.insight
			if key == "" {
				key = e.reason
			}
			if key != "" {
				failureReasons[key]++
			}
		}
		if e.status == "success" {
			key := e.insight
			if key == "" {
				key = e.mission
			}
			if key != "" {
				successPatterns[key]++
			}
		}
		if e.highSeverity != "" {
			highSeverityAlerts[e.highSeverity]++
		}
	}

	var luminas []paintress.Lumina

	// HIGH severity D-Mail alerts become immediate Luminas (threshold = 1)
	for names, count := range highSeverityAlerts {
		luminas = append(luminas, paintress.Lumina{
			Pattern: fmt.Sprintf("[ALERT] HIGH severity D-Mail in past expedition: %s", names),
			Source:  "high-severity-alert",
			Uses:    count,
		})
	}

	// Failures that repeat become defensive Luminas
	for reason, count := range failureReasons {
		if count >= 2 {
			luminas = append(luminas, paintress.Lumina{
				Pattern: fmt.Sprintf("[WARN] Avoid — failed %d times: %s", count, reason),
				Source:  "failure-pattern",
				Uses:    count,
			})
		}
	}

	// Successful patterns become offensive Luminas
	for pattern, count := range successPatterns {
		if count >= 3 {
			luminas = append(luminas, paintress.Lumina{
				Pattern: fmt.Sprintf("[OK] Proven approach (%dx successful): %s", count, pattern),
				Source:  "success-pattern",
				Uses:    count,
			})
		}
	}

	return luminas
}

func extractValue(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	val := strings.TrimSpace(parts[1])
	val = strings.TrimPrefix(val, "**")
	val = strings.TrimSuffix(val, "**")
	return val
}
