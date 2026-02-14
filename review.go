package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const maxReviewCycles = 3

// ReviewResult holds the outcome of a code review execution.
type ReviewResult struct {
	Passed   bool   // true if no actionable comments were found
	Output   string // raw review output
	Comments string // extracted review comments (empty if passed)
}

// RunReview executes the review command and parses the output.
// It returns a ReviewResult indicating whether the review passed
// and any comments that were found.
func RunReview(ctx context.Context, reviewCmd string, dir string) (*ReviewResult, error) {
	if strings.TrimSpace(reviewCmd) == "" {
		return &ReviewResult{Passed: true}, nil
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", reviewCmd)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	output := string(out)

	// Rate limit / quota signals — skip review gracefully regardless of exit code
	if isRateLimited(output) {
		return nil, fmt.Errorf("review service rate/quota limited")
	}

	// Check for review comments before treating non-zero exit as failure.
	// Many review tools exit non-zero when they find issues.
	if hasReviewComments(output) {
		return &ReviewResult{
			Passed:   false,
			Output:   output,
			Comments: output,
		}, nil
	}

	// Non-zero exit with no review comments — command failure, treat as skippable
	if err != nil {
		return nil, fmt.Errorf("review command failed: %w\noutput: %s", err, summarizeReview(output))
	}

	return &ReviewResult{
		Passed: true,
		Output: output,
	}, nil
}

// hasReviewComments checks whether the review output contains actionable comments.
// It looks for priority tags [P0]–[P4] (P0 = critical, P4 = nit) commonly used by
// code review tools like Codex CLI, or the "Review comment:" keyword pattern.
func hasReviewComments(output string) bool {
	indicators := []string{"[P0]", "[P1]", "[P2]", "[P3]", "[P4]"}
	for _, tag := range indicators {
		if strings.Contains(output, tag) {
			return true
		}
	}
	// Also check for "Review comment:" pattern
	if strings.Contains(output, "Review comment") {
		return true
	}
	return false
}

// isRateLimited checks whether the review output indicates a rate or quota limit.
func isRateLimited(output string) bool {
	lower := strings.ToLower(output)
	signals := []string{
		"rate limit",
		"rate_limit",
		"quota exceeded",
		"quota limit",
		"too many requests",
		"429",
		"usage limit",
	}
	for _, s := range signals {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// BuildReviewFixPrompt creates a focused prompt for fixing review comments.
// This is a lightweight invocation — no Lumina, Gradient, or mission context needed.
func BuildReviewFixPrompt(branch string, comments string) string {
	return fmt.Sprintf(`You are on branch %s with an open PR. A code review found the following issues:

%s

Fix all review comments above. Commit and push your changes. Do not create a new branch or PR.
Keep fixes focused — only address the review comments, do not refactor unrelated code.
If a review comment is unclear or you cannot resolve it after a reasonable attempt, skip it and move on to the next.
Do not change the Linear issue status — it stays in its current state until the next Expedition.`, branch, comments)
}

// summarizeReview normalizes multi-line review output to a single line
// and truncates for journal storage. Uses rune-based truncation to avoid
// splitting multi-byte UTF-8 characters.
func summarizeReview(comments string) string {
	// Normalize to single line for journal compatibility
	normalized := strings.Join(strings.Fields(comments), " ")

	const maxLen = 500
	runes := []rune(normalized)
	if len(runes) <= maxLen {
		return normalized
	}
	return string(runes[:maxLen]) + "...(truncated)"
}
