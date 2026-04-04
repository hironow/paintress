package domain

import (
	"sort"
	"strings"
)

// ReviewComment represents a single extracted review comment with its priority.
type ReviewComment struct {
	Priority int    // 0 = highest (P0), 4 = lowest (P4)
	Text     string // the comment text including the priority tag
}

// priorityTags is the ordered set of supported priority tags.
var priorityTags = []struct {
	tag      string
	priority int
}{
	{"[P0]", 0},
	{"[P1]", 1},
	{"[P2]", 2},
	{"[P3]", 3},
	{"[P4]", 4},
}

// detectPriority returns the numeric priority of a line based on its priority tag.
// Returns -1 if no priority tag is found.
func detectPriority(line string) int {
	for _, pt := range priorityTags {
		if strings.Contains(line, pt.tag) {
			return pt.priority
		}
	}
	return -1
}

// extractReviewComments parses review output and extracts structured ReviewComment values.
// Lines containing priority tags [P0]–[P4] are extracted and sorted by priority (P0 first).
// If no priority tags are found but the output contains "Review comment", the full output
// is returned as a single raw fallback comment with priority 4.
// Returns an empty slice when the output contains no recognizable review content.
func extractReviewComments(output string) []ReviewComment {
	if strings.TrimSpace(output) == "" {
		return nil
	}

	lines := strings.Split(output, "\n")
	var comments []ReviewComment

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		p := detectPriority(trimmed)
		if p >= 0 {
			comments = append(comments, ReviewComment{Priority: p, Text: trimmed})
		}
	}

	if len(comments) > 0 {
		sort.SliceStable(comments, func(i, j int) bool {
			return comments[i].Priority < comments[j].Priority
		})
		return comments
	}

	// Fallback: if "Review comment" keyword is present, return full output as one entry.
	if strings.Contains(output, "Review comment") {
		return []ReviewComment{{Priority: 4, Text: strings.TrimSpace(output)}}
	}

	return nil
}
