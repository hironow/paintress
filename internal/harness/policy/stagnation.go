package policy

import "strings"

// CountPriorityTags counts the number of priority tags ([P0] through [P4])
// in review output. Each occurrence is counted individually.
func CountPriorityTags(output string) int {
	tags := []string{"[P0]", "[P1]", "[P2]", "[P3]", "[P4]"}
	count := 0
	for _, tag := range tags {
		count += strings.Count(output, tag)
	}
	return count
}

// IsStagnant reports whether the review loop has stagnated by comparing
// the current cycle's tag count against the previous cycle's count.
// Stagnation is defined as no decrease in comment count.
// Returns false when previousCount is zero (no baseline available).
// Returns false when both counts are zero (issue is resolved).
func IsStagnant(currentCount, previousCount int) bool {
	if previousCount == 0 {
		return false
	}
	return currentCount >= previousCount
}
