package domain

import "path/filepath"

func JournalDir(continent string) string {
	return filepath.Join(continent, StateDir, "journal")
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
