package domain

import (
	"fmt"
	"strings"
)

// ReviewGateStatus holds the outcome of the review gate for PR body reporting.
type ReviewGateStatus struct {
	Passed       bool
	Skipped      bool
	Cycle        int
	MaxCycles    int
	LastComments string
}

// FormatSection returns a Markdown section for the PR body.
func (s ReviewGateStatus) FormatSection() string {
	var sb strings.Builder
	sb.WriteString("## Review Gate\n\n")

	if s.Skipped {
		sb.WriteString("- Status: **SKIPPED** (no review command configured)\n")
		return sb.String()
	}

	if s.Passed {
		sb.WriteString(fmt.Sprintf("- Status: **PASSED** (cycle %d/%d, 0 comments remaining)\n", s.Cycle, s.MaxCycles))
	} else {
		sb.WriteString(fmt.Sprintf("- Status: **NOT RESOLVED** (%d/%d cycles exhausted, comments remaining)\n", s.Cycle, s.MaxCycles))
	}

	return sb.String()
}

const reviewGateHeader = "## Review Gate"

// AppendReviewGateSection appends or replaces the Review Gate section in a PR body.
func AppendReviewGateSection(body, section string) string {
	idx := strings.Index(body, reviewGateHeader)
	if idx == -1 {
		return strings.TrimRight(body, "\n") + "\n\n" + section
	}
	// Find the end of the existing section (next ## or end of body)
	rest := body[idx+len(reviewGateHeader):]
	nextHeader := strings.Index(rest, "\n## ")
	if nextHeader == -1 {
		return body[:idx] + section
	}
	return strings.TrimRight(body[:idx]+section, "\n") + "\n" + rest[nextHeader:]
}
