package paintress

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

type ExpeditionReport struct {
	Expedition  int
	IssueID     string
	IssueTitle  string
	MissionType string
	Branch      string
	PRUrl       string
	Status      string
	Reason      string
	Remaining   string
	BugsFound   int
	BugIssues   string
	Insight     string
	FailureType string
}

func ParseReport(output string, expNum int) (*ExpeditionReport, ReportStatus) {
	if strings.Contains(output, "__EXPEDITION_COMPLETE__") {
		return &ExpeditionReport{Expedition: expNum}, StatusComplete
	}

	startMarker := "__EXPEDITION_REPORT__"
	endMarker := "__EXPEDITION_END__"
	startIdx := strings.Index(output, startMarker)
	endIdx := strings.Index(output, endMarker)

	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
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
