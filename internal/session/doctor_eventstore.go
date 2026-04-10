package session

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// checkDeadLetters reports outbox items that have exceeded max retry count.
func checkDeadLetters(ctx context.Context, continent string) domain.DoctorCheck {
	// Check DB file exists before opening (avoid creating dirs/DB as side effect)
	dbPath := filepath.Join(continent, domain.StateDir, ".run", "outbox.db")
	if _, err := os.Stat(dbPath); err != nil {
		return domain.DoctorCheck{
			Name:    "dead-letters",
			Status:  domain.CheckSkip,
			Message: "no outbox DB",
		}
	}
	store, err := NewOutboxStoreForDir(continent)
	if err != nil {
		return domain.DoctorCheck{
			Name:    "dead-letters",
			Status:  domain.CheckSkip,
			Message: "outbox store unavailable",
		}
	}
	defer store.Close()
	count, err := store.DeadLetterCount(ctx)
	if err != nil {
		return domain.DoctorCheck{
			Name:    "dead-letters",
			Status:  domain.CheckSkip,
			Message: fmt.Sprintf("dead letter count: %v", err),
		}
	}
	if count > 0 {
		return domain.DoctorCheck{
			Name:    "dead-letters",
			Status:  domain.CheckWarn,
			Message: fmt.Sprintf("%d dead-lettered outbox item(s)", count),
			Hint:    "run 'paintress dead-letters purge --execute' to remove",
		}
	}
	return domain.DoctorCheck{
		Name:    "dead-letters",
		Status:  domain.CheckOK,
		Message: "no dead-lettered items",
	}
}

// checkEventStore verifies that event JSONL files are parseable using the same
// json.Unmarshal judgment as the real event store replay. This catches both
// syntactic corruption (invalid JSON) and structural corruption (valid JSON but
// incompatible with domain.Event, e.g. bad timestamp format).
// Scans .expedition/events/*.jsonl files. // nosemgrep: layer-session-no-event-persistence [permanent]
// Returns a Warning-level check.
func checkEventStore(continent string) domain.DoctorCheck {
	eventsDir := filepath.Join(continent, domain.StateDir, "events")
	entries, err := os.ReadDir(eventsDir)
	if err != nil {
		return domain.DoctorCheck{
			Name:    "events",
			Status:  domain.CheckWarn,
			Message: "events/ not found",
			Hint:    `run "paintress init <repo-path>" to create events directory`,
		}
	}
	var files, lines, corruptLines int
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") { // nosemgrep: layer-session-no-event-persistence [permanent]
			continue
		}
		f, err := os.Open(filepath.Join(eventsDir, entry.Name()))
		if err != nil {
			return domain.DoctorCheck{
				Name:    "events",
				Status:  domain.CheckWarn,
				Message: "read error: " + err.Error(),
				Hint:    "check file permissions on .expedition/events/",
			}
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var ev domain.Event
			if jsonErr := json.Unmarshal([]byte(line), &ev); jsonErr != nil {
				corruptLines++
				continue
			}
			lines++
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return domain.DoctorCheck{
				Name:    "events",
				Status:  domain.CheckWarn,
				Message: "scan error: " + err.Error(),
				Hint:    "check file permissions on .expedition/events/",
			}
		}
		files++
	}
	if files == 0 {
		return domain.DoctorCheck{
			Name:    "events",
			Status:  domain.CheckWarn,
			Message: "no .jsonl files found", // nosemgrep: layer-session-no-event-persistence [permanent]
		}
	}
	if corruptLines > 0 {
		return domain.DoctorCheck{
			Name:    "events",
			Status:  domain.CheckWarn,
			Message: fmt.Sprintf("%d corrupt line(s) in event store (%d file(s), %d valid events)", corruptLines, files, lines),
			Hint:    "corrupt lines are skipped during replay — review JSONL files in " + eventsDir,
		}
	}
	return domain.DoctorCheck{
		Name:    "events",
		Status:  domain.CheckOK,
		Message: fmt.Sprintf("%s (%d files, %d events OK)", eventsDir, files, lines),
	}
}
