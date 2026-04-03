package policy

import (
	"fmt"
	"strings"
)

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

// ExpandReviewCmd replaces placeholders in the review command string.
// Supported placeholders:
//
//	{file}   → review working directory
//	{branch} → current git branch name
func ExpandReviewCmd(cmd, dir, branch string) string {
	cmd = strings.ReplaceAll(cmd, "{file}", dir)
	cmd = strings.ReplaceAll(cmd, "{branch}", branch)
	return cmd
}

// BuildReviewFixPrompt creates a focused prompt for fixing review comments.
func BuildReviewFixPrompt(branch string, comments string) string {
	return fmt.Sprintf(`You are on branch %s with an open PR. A code review found the following issues:

%s

Fix all review comments above. Commit and push your changes. Do not create a new branch or PR.
Keep fixes focused — only address the review comments, do not refactor unrelated code.
If a review comment is unclear or you cannot resolve it after a reasonable attempt, skip it and move on to the next.
Do not change the Linear issue status — it stays in its current state until the next Expedition.`, branch, comments)
}

// SummarizeReview normalizes multi-line review output and truncates.
func SummarizeReview(comments string) string {
	normalized := strings.Join(strings.Fields(comments), " ")
	const maxLen = 500
	runes := []rune(normalized)
	if len(runes) <= maxLen {
		return normalized
	}
	return string(runes[:maxLen]) + "...(truncated)"
}
