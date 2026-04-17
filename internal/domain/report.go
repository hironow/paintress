package domain

import (
	"fmt"
	"strings"
)

type ReportStatus int

const (
	StatusSuccess ReportStatus = iota
	StatusSkipped
	StatusFailed
	StatusComplete
	StatusParseError
)

type ExpeditionReport struct { // nosemgrep: domain-primitives.public-string-field-go -- PRUrl is a plain record field; newtype adds no safety benefit [permanent]
	Expedition         int
	IssueID            string
	IssueTitle         string
	MissionType        string
	Branch             string
	PRUrl              string
	Status             string
	Reason             string
	Remaining          string
	BugsFound          int
	BugIssues          string
	Insight            string
	FailureType        string
	HighSeverityDMails string // comma-separated names of mid-expedition HIGH severity d-mails

	// Wave-centric mode fields (empty in Linear mode)
	WaveID string
	StepID string
}

// PRIndexEntry represents a single PR URL entry extracted from an expedition report.
type PRIndexEntry struct { // nosemgrep: domain-primitives.public-string-field-go -- PRUrl is a plain record field; newtype adds no safety benefit [permanent]
	Expedition int    `json:"expedition"`
	IssueID    string `json:"issue_id"`
	PRUrl      string `json:"pr_url"`
}

// ExtractPRURLs collects PR URLs from expedition reports, filtering out empty
// and "none" values. Returns entries in input order.
func ExtractPRURLs(reports []*ExpeditionReport) []PRIndexEntry {
	var entries []PRIndexEntry
	for _, r := range reports {
		if r == nil || r.PRUrl == "" || r.PRUrl == "none" {
			continue
		}
		entries = append(entries, PRIndexEntry{
			Expedition: r.Expedition,
			IssueID:    r.IssueID,
			PRUrl:      r.PRUrl,
		})
	}
	return entries
}

func ParseReport(output string, expNum int) (*ExpeditionReport, ReportStatus) {
	if strings.Contains(output, "__EXPEDITION_COMPLETE__") {
		return &ExpeditionReport{Expedition: expNum}, StatusComplete
	}

	startMarker := "__EXPEDITION_REPORT__"
	startIdx := strings.Index(output, startMarker)
	if startIdx == -1 {
		return nil, StatusParseError
	}

	// Accept both canonical and commonly hallucinated end markers.
	endIdx := strings.Index(output, "__EXPEDITION_END__")
	if alt := strings.Index(output, "__END_EXPEDITION_REPORT__"); alt != -1 && (endIdx == -1 || alt < endIdx) {
		endIdx = alt
	}
	if endIdx == -1 || endIdx <= startIdx {
		return nil, StatusParseError
	}

	block := output[startIdx+len(startMarker) : endIdx]
	report := &ExpeditionReport{Expedition: expNum}

	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "issue_id":
			report.IssueID = val
		case "issue_title":
			report.IssueTitle = val
		case "mission_type":
			report.MissionType = val
		case "branch":
			report.Branch = val
		case "pr_url":
			report.PRUrl = val
		case "status":
			report.Status = val
		case "reason":
			report.Reason = val
		case "remaining_issues":
			report.Remaining = val
		case "bugs_found":
			fmt.Sscanf(val, "%d", &report.BugsFound)
		case "bug_issues":
			report.BugIssues = val
		case "insight":
			report.Insight = val
		case "failure_type":
			report.FailureType = val
		}
	}

	switch report.Status {
	case "success":
		return report, StatusSuccess
	case "skipped":
		return report, StatusSkipped
	case "failed":
		return report, StatusFailed
	default:
		return report, StatusParseError
	}
}
