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

// ReviewCycleHistory tracks comment counts across review cycles to detect diminishing returns.
type ReviewCycleHistory struct {
	InitialCommentCount int
	FinalCommentCount   int
	CycleCount          int
}

// ImprovementRate returns (initial-final)/initial.
// Returns 0.0 when InitialCommentCount is zero.
func (h ReviewCycleHistory) ImprovementRate() float64 {
	if h.InitialCommentCount == 0 {
		return 0.0
	}
	return float64(h.InitialCommentCount-h.FinalCommentCount) / float64(h.InitialCommentCount)
}

// IsStalled returns true when CycleCount >= stallWindow and ImprovementRate is zero.
func (h ReviewCycleHistory) IsStalled(stallWindow int) bool {
	if h.CycleCount < stallWindow {
		return false
	}
	return h.ImprovementRate() == 0.0
}

// FormatStallWarning returns a human-readable warning about review cycle stall.
func (h ReviewCycleHistory) FormatStallWarning() string {
	return fmt.Sprintf(
		"Stall detected: %d review cycles with no improvement (rate=%.2f, comments: %d → %d)",
		h.CycleCount,
		h.ImprovementRate(),
		h.InitialCommentCount,
		h.FinalCommentCount,
	)
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
