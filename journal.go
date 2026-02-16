package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func JournalDir(continent string) string {
	return filepath.Join(continent, ".expedition", "journal")
}

// JournalEntry is a structured journal record.
type JournalEntry struct {
	Expedition  int
	Date        string
	IssueID     string
	IssueTitle  string
	MissionType string
	Status      string
	Reason      string
	PRUrl       string
	BugsFound   int
	BugIssues   string
}

func WriteJournal(continent string, report *ExpeditionReport) error {
	dir := JournalDir(continent)
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
	)

	return os.WriteFile(path, []byte(content), 0644)
}

// ListJournalFiles returns journal file paths sorted by name (ascending).
func ListJournalFiles(continent string) ([]string, error) {
	dir := JournalDir(continent)
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
