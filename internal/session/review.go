package session

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

const maxReviewGateCycles = 3

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
			if domain.IsRateLimited(output) {
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
		return nil, fmt.Errorf("review command failed: %w\noutput: %s", err, domain.SummarizeReview(output))
	}

	// Exit 0 → pass, regardless of output content.
	return &ReviewResult{
		Passed: true,
		Output: output,
	}, nil
}
