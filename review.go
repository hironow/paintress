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
	if reviewCmd == "" {
		return &ReviewResult{Passed: true}, nil
	}

	parts := strings.Fields(reviewCmd)
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		// Rate limit, quota limit, or command failure — treat as skippable
		return nil, fmt.Errorf("review command failed: %w\noutput: %s", err, output)
	}

	// Rate limit / quota signals in successful output (command exited 0 but reported limit)
	if isRateLimited(output) {
		return nil, fmt.Errorf("review service rate/quota limited")
	}

	// Detect review comments by priority tags like [P1], [P2], etc.
	if hasReviewComments(output) {
		return &ReviewResult{
			Passed:   false,
			Output:   output,
			Comments: output,
		}, nil
	}

	return &ReviewResult{
		Passed: true,
		Output: output,
	}, nil
}

// hasReviewComments checks whether the review output contains actionable comments.
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
Keep fixes focused — only address the review comments, do not refactor unrelated code.`, branch, comments)
}

// summarizeReview truncates long review output for journal storage.
func summarizeReview(comments string) string {
	const maxLen = 500
	if len(comments) <= maxLen {
		return comments
	}
	return comments[:maxLen] + "...(truncated)"
}
