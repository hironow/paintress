package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/hironow/paintress"
)

// WriteJournal writes an expedition report to the journal directory.
func WriteJournal(continent string, report *paintress.ExpeditionReport) error {
	dir := paintress.JournalDir(continent)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("%03d.md", report.Expedition)
	path := filepath.Join(dir, filename)

	content := fmt.Sprintf(`# Expedition #%d — Journal
# This is a record of a past Expedition. Use the Insight field as a lesson for your mission.

- **Date**: %s
- **Issue**: %s — %s
- **Mission**: %s
- **Status**: %s
- **Reason**: %s
- **PR**: %s
- **Bugs found**: %d
- **Bug issues**: %s
- **Insight**: %s
- **Failure type**: %s
- **HIGH severity D-Mail**: %s
`,
		report.Expedition,
		time.Now().Format("2006-01-02 15:04:05"),
		report.IssueID, report.IssueTitle,
		report.MissionType,
		report.Status,
		report.Reason,
		report.PRUrl,
		report.BugsFound,
		report.BugIssues,
		report.Insight,
		report.FailureType,
		report.HighSeverityDMails,
	)

	return os.WriteFile(path, []byte(content), 0644)
}

// WritePRIndex appends a PR URL index entry to the pr-index.jsonl file
// in the journal directory. Skips entries with empty or "none" PR URLs.
func WritePRIndex(continent string, report *paintress.ExpeditionReport) error {
	if report.PRUrl == "" || report.PRUrl == "none" {
		return nil
	}
	dir := paintress.JournalDir(continent)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	entry := paintress.PRIndexEntry{
		Expedition: report.Expedition,
		IssueID:    report.IssueID,
		PRUrl:      report.PRUrl,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("pr index: marshal: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(dir, "pr-index.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("pr index: open: %w", err)
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// ListJournalFiles returns journal file paths sorted by name (ascending).
func ListJournalFiles(continent string) ([]string, error) {
	dir := paintress.JournalDir(continent)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" && e.Name() != "000.md" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}
