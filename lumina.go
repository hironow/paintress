package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Lumina represents a learned passive skill extracted from past expedition journals.
// In the game, Pictos are mastered after 4 uses, unlocking their Lumina for all characters.
// Here, patterns that recur across journals become Luminas injected into future prompts.
type Lumina struct {
	Pattern string // The learned pattern / lesson
	Source  string // Which journal(s) contributed
	Uses    int    // How many times this pattern appeared
}

// LuminaPath returns the path to the lumina file on the Continent.
func LuminaPath(continent string) string {
	return filepath.Join(continent, ".expedition", "lumina.md")
}

// ScanJournalsForLumina reads all journal files in parallel goroutines,
// extracts failure reasons and success patterns, and returns Luminas.
// This runs at each Expedition Flag (before departure), just like
// resting at a flag lets you learn new skills.
func ScanJournalsForLumina(continent string) []Lumina {
	files, err := ListJournalFiles(continent)
	if err != nil || len(files) == 0 {
		return nil
	}

	type journalData struct {
		status  string
		reason  string
		mission string
		issue   string
	}

	// Parallel journal scanning
	var mu sync.Mutex
	var wg sync.WaitGroup
	var entries []journalData

	for _, f := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			content, err := os.ReadFile(path)
			if err != nil {
				return
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
				}
			}

			mu.Lock()
			entries = append(entries, entry)
			mu.Unlock()
		}(f)
	}
	wg.Wait()

	// Aggregate patterns
	failureReasons := make(map[string]int)
	successPatterns := make(map[string]int)

	for _, e := range entries {
		if e.status == "failed" && e.reason != "" {
			failureReasons[e.reason]++
		}
		if e.status == "success" && e.mission != "" {
			successPatterns[e.mission]++
		}
	}

	var luminas []Lumina

	// Failures that repeat become defensive Luminas (like parry skills)
	for reason, count := range failureReasons {
		if count >= 2 { // "Mastered" after 2 occurrences (like Pictos after 4 uses, scaled down)
			luminas = append(luminas, Lumina{
				Pattern: fmt.Sprintf("[WARN] Failed %d times: %s", count, reason),
				Source:  "failure-pattern",
				Uses:    count,
			})
		}
	}

	// Successful mission types become offensive Luminas
	for mission, count := range successPatterns {
		if count >= 3 { // Reliable pattern after 3 successes
			luminas = append(luminas, Lumina{
				Pattern: fmt.Sprintf("[OK] %s mission: %d proven successes", mission, count),
				Source:  "success-pattern",
				Uses:    count,
			})
		}
	}

	return luminas
}

// WriteLumina writes the current Lumina state to the Continent.
func WriteLumina(continent string, luminas []Lumina) error {
	if len(luminas) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString(Msg("lumina_header"))
	sb.WriteString("\n")

	sb.WriteString(Msg("lumina_defensive"))
	sb.WriteString("\n")
	for _, l := range luminas {
		if l.Source == "failure-pattern" {
			sb.WriteString(fmt.Sprintf("- %s\n", l.Pattern))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(Msg("lumina_offensive"))
	sb.WriteString("\n")
	for _, l := range luminas {
		if l.Source == "success-pattern" {
			sb.WriteString(fmt.Sprintf("- %s\n", l.Pattern))
		}
	}

	return os.WriteFile(LuminaPath(continent), []byte(sb.String()), 0644)
}

// FormatLuminaForPrompt formats Luminas for injection into the expedition prompt.
func FormatLuminaForPrompt(luminas []Lumina) string {
	if len(luminas) == 0 {
		return Msg("lumina_none")
	}

	var sb strings.Builder
	for _, l := range luminas {
		sb.WriteString(fmt.Sprintf("- %s\n", l.Pattern))
	}
	return sb.String()
}

func extractValue(line string) string {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) < 2 {
		return ""
	}
	val := strings.TrimSpace(parts[1])
	// Remove markdown bold markers
	val = strings.TrimPrefix(val, "**")
	val = strings.TrimSuffix(val, "**")
	return val
}
