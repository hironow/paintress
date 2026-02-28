package session

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const maxReviewCycles = 3

// minReviewTimeout is the floor for the per-cycle review timeout.
var minReviewTimeout = 30 * time.Second

// gitCmdTimeout is the per-call timeout for git operations in the review loop.
var gitCmdTimeout = 30 * time.Second

// ReviewResult holds the outcome of a code review execution.
type ReviewResult struct {
	Passed   bool   // true if no actionable comments were found
	Output   string // raw review output
	Comments string // extracted review comments (empty if passed)
}

// RunReview executes the review command and parses the output.
func RunReview(ctx context.Context, reviewCmd string, dir string) (*ReviewResult, error) {
	if strings.TrimSpace(reviewCmd) == "" {
		return &ReviewResult{Passed: true}, nil
	}

	cmd := exec.CommandContext(ctx, shellName(), shellFlag(), reviewCmd)
	cmd.Dir = dir
	cmd.WaitDelay = 1 * time.Second

	out, err := cmd.CombinedOutput()
	output := string(out)

	// Context cancellation (timeout, signal) is infrastructure, not review result.
	if ctx.Err() != nil {
		return nil, fmt.Errorf("review command canceled: %w", ctx.Err())
	}

	// Exit code is the primary signal (P1-8: exit code based detection).
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Rate limit on non-zero exit is a service error, not review comments.
			if isRateLimited(output) {
				return nil, fmt.Errorf("review service rate/quota limited")
			}
			// Non-zero exit → review found comments.
			return &ReviewResult{
				Passed:   false,
				Output:   output,
				Comments: output,
			}, nil
		}
		// Non-exit errors (failed to start, etc.)
		return nil, fmt.Errorf("review command failed: %w\noutput: %s", err, summarizeReview(output))
	}

	// Exit 0 → pass, regardless of output content.
	return &ReviewResult{
		Passed: true,
		Output: output,
	}, nil
}

func hasReviewComments(output string) bool {
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

func isRateLimited(output string) bool {
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

// summarizeReview normalizes multi-line review output and truncates.
func summarizeReview(comments string) string {
	normalized := strings.Join(strings.Fields(comments), " ")
	const maxLen = 500
	runes := []rune(normalized)
	if len(runes) <= maxLen {
		return normalized
	}
	return string(runes[:maxLen]) + "...(truncated)"
}
