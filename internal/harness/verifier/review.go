package verifier

import "strings"

// HasReviewComments reports whether the review output contains actionable
// review comment indicators (priority tags or the "Review comment" keyword).
func HasReviewComments(output string) bool {
	indicators := []string{"[P0]", "[P1]", "[P2]", "[P3]", "[P4]"}
	for _, tag := range indicators {
		if strings.Contains(output, tag) {
			return true
		}
	}
	if strings.Contains(output, "Review comment") {
		return true
	}
	return false
}

// IsRateLimited reports whether the output contains rate/quota limiting signals.
func IsRateLimited(output string) bool {
	lower := strings.ToLower(output)
	signals := []string{
		"rate limit",
		"rate_limit",
		"quota exceeded",
		"quota limit",
		"too many requests",
		"usage limit",
	}
	for _, s := range signals {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
