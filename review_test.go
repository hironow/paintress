package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// === hasReviewComments ===

func TestHasReviewComments_P0(t *testing.T) {
	if !hasReviewComments("[P0] Critical: missing null check") {
		t.Error("should detect [P0]")
	}
}

func TestHasReviewComments_P1(t *testing.T) {
	if !hasReviewComments("[P1] Reset live URL state") {
		t.Error("should detect [P1]")
	}
}

func TestHasReviewComments_P2(t *testing.T) {
	if !hasReviewComments("[P2] Some review finding") {
		t.Error("should detect [P2]")
	}
}

func TestHasReviewComments_P3(t *testing.T) {
	if !hasReviewComments("[P3] Minor style issue") {
		t.Error("should detect [P3]")
	}
}

func TestHasReviewComments_P4(t *testing.T) {
	if !hasReviewComments("[P4] Nitpick") {
		t.Error("should detect [P4]")
	}
}

func TestHasReviewComments_ReviewCommentKeyword(t *testing.T) {
	if !hasReviewComments("Review comment:\n- Fix the return type") {
		t.Error("should detect 'Review comment' keyword")
	}
}

func TestHasReviewComments_NoComments(t *testing.T) {
	if hasReviewComments("I did not find any discrete, actionable regressions in the changed application code.") {
		t.Error("clean review should not be detected as having comments")
	}
}

func TestHasReviewComments_EmptyOutput(t *testing.T) {
	if hasReviewComments("") {
		t.Error("empty output should not be detected as having comments")
	}
}

func TestHasReviewComments_MultipleTags(t *testing.T) {
	output := "[P1] First issue\n[P2] Second issue\n[P3] Third issue"
	if !hasReviewComments(output) {
		t.Error("should detect multiple priority tags")
	}
}

// === isRateLimited ===

func TestIsRateLimited_RateLimit(t *testing.T) {
	if !isRateLimited("Error: rate limit exceeded, try again later") {
		t.Error("should detect 'rate limit'")
	}
}

func TestIsRateLimited_RateLimitUnderscore(t *testing.T) {
	if !isRateLimited("rate_limit_error: 429") {
		t.Error("should detect 'rate_limit'")
	}
}

func TestIsRateLimited_QuotaExceeded(t *testing.T) {
	if !isRateLimited("quota exceeded for this billing period") {
		t.Error("should detect 'quota exceeded'")
	}
}

func TestIsRateLimited_QuotaLimit(t *testing.T) {
	if !isRateLimited("quota limit reached") {
		t.Error("should detect 'quota limit'")
	}
}

func TestIsRateLimited_TooManyRequests(t *testing.T) {
	if !isRateLimited("too many requests") {
		t.Error("should detect 'too many requests'")
	}
}

func TestIsRateLimited_429(t *testing.T) {
	if !isRateLimited("HTTP 429: slow down") {
		t.Error("should detect '429'")
	}
}

func TestIsRateLimited_UsageLimit(t *testing.T) {
	if !isRateLimited("usage limit hit") {
		t.Error("should detect 'usage limit'")
	}
}

func TestIsRateLimited_CaseInsensitive(t *testing.T) {
	if !isRateLimited("RATE LIMIT exceeded") {
		t.Error("should detect rate limit case-insensitively")
	}
}

func TestIsRateLimited_NormalOutput(t *testing.T) {
	if isRateLimited("I did not find any discrete regressions.") {
		t.Error("normal output should not be rate limited")
	}
}

func TestIsRateLimited_EmptyOutput(t *testing.T) {
	if isRateLimited("") {
		t.Error("empty output should not be rate limited")
	}
}

// === summarizeReview ===

func TestSummarizeReview_Short(t *testing.T) {
	input := "Short review comment"
	got := summarizeReview(input)
	if got != input {
		t.Errorf("short input should be unchanged, got %q", got)
	}
}

func TestSummarizeReview_ExactlyMaxLen(t *testing.T) {
	input := strings.Repeat("a", 500)
	got := summarizeReview(input)
	if got != input {
		t.Error("exactly 500 chars should not be truncated")
	}
}

func TestSummarizeReview_OverMaxLen(t *testing.T) {
	input := strings.Repeat("a", 600)
	got := summarizeReview(input)
	if len(got) > 520 {
		t.Errorf("truncated output too long: %d", len(got))
	}
	if !strings.HasSuffix(got, "...(truncated)") {
		t.Error("truncated output should end with '...(truncated)'")
	}
}

func TestSummarizeReview_Empty(t *testing.T) {
	got := summarizeReview("")
	if got != "" {
		t.Errorf("empty input should return empty, got %q", got)
	}
}

// === BuildReviewFixPrompt ===

func TestBuildReviewFixPrompt_ContainsBranch(t *testing.T) {
	prompt := BuildReviewFixPrompt("feat/my-branch", "[P1] Fix null check")
	if !strings.Contains(prompt, "feat/my-branch") {
		t.Error("prompt should contain the branch name")
	}
}

func TestBuildReviewFixPrompt_ContainsComments(t *testing.T) {
	comments := "[P2] Reset live URL state when entering demo mode"
	prompt := BuildReviewFixPrompt("feat/demo", comments)
	if !strings.Contains(prompt, comments) {
		t.Error("prompt should contain the review comments")
	}
}

func TestBuildReviewFixPrompt_NoBranchCreation(t *testing.T) {
	prompt := BuildReviewFixPrompt("feat/x", "fix this")
	if !strings.Contains(prompt, "Do not create a new branch") {
		t.Error("prompt should instruct not to create new branch")
	}
}

func TestBuildReviewFixPrompt_NoStatusChange(t *testing.T) {
	prompt := BuildReviewFixPrompt("feat/x", "fix this")
	if strings.Contains(prompt, "Testing") {
		t.Error("prompt should NOT instruct to change status to Testing")
	}
	if !strings.Contains(prompt, "Do not change the Linear issue status") {
		t.Error("prompt should explicitly say not to change issue status")
	}
}

// === RunReview ===

func TestRunReview_EmptyCommand(t *testing.T) {
	// given
	ctx := context.Background()

	// when
	result, err := RunReview(ctx, "", "/tmp")

	// then
	if err != nil {
		t.Fatalf("empty command should not error: %v", err)
	}
	if !result.Passed {
		t.Error("empty command should auto-pass")
	}
}

func TestRunReview_WhitespaceOnlyCommand(t *testing.T) {
	// given
	ctx := context.Background()

	// when
	result, err := RunReview(ctx, "   ", "/tmp")

	// then
	if err != nil {
		t.Fatalf("whitespace-only command should not error: %v", err)
	}
	if !result.Passed {
		t.Error("whitespace-only command should auto-pass")
	}
}

func TestRunReview_NonZeroExitWithComments(t *testing.T) {
	// given — review tool exits non-zero when it finds issues (common for CI/lint tools)
	ctx := context.Background()
	dir := t.TempDir()

	// Create a script that outputs review comments and exits non-zero
	scriptPath := filepath.Join(dir, "review.sh")
	os.WriteFile(scriptPath, []byte("#!/bin/bash\necho '[P2] Reset live URL'\nexit 1\n"), 0755)

	// when
	result, err := RunReview(ctx, scriptPath, dir)

	// then — should return review comments, NOT an error
	if err != nil {
		t.Fatalf("non-zero exit with review comments should not error: %v", err)
	}
	if result.Passed {
		t.Error("review with [P2] comments should not pass")
	}
	if result.Comments == "" {
		t.Error("comments should not be empty")
	}
}

func TestRunReview_CommandNotFound(t *testing.T) {
	// given
	ctx := context.Background()

	// when
	_, err := RunReview(ctx, "nonexistent-review-tool-xyz --check", t.TempDir())

	// then
	if err == nil {
		t.Error("non-existent command should return error")
	}
}

func TestRunReview_PassingReview(t *testing.T) {
	// given
	ctx := context.Background()
	dir := t.TempDir()

	// when — echo outputs clean review text
	result, err := RunReview(ctx, "echo I did not find any discrete actionable regressions", dir)

	// then
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if !result.Passed {
		t.Error("clean review should pass")
	}
}

func TestRunReview_FailingReview(t *testing.T) {
	// given
	ctx := context.Background()
	dir := t.TempDir()

	// when — echo outputs review with comments
	result, err := RunReview(ctx, "echo [P2] Reset live URL state", dir)

	// then
	if err != nil {
		t.Fatalf("should not error: %v", err)
	}
	if result.Passed {
		t.Error("review with [P2] comments should not pass")
	}
	if result.Comments == "" {
		t.Error("comments should not be empty")
	}
}

func TestRunReview_RateLimitInOutput(t *testing.T) {
	// given
	ctx := context.Background()
	dir := t.TempDir()

	// when — echo outputs rate limit message (exit 0 but rate limited)
	_, err := RunReview(ctx, "echo rate limit exceeded", dir)

	// then
	if err == nil {
		t.Error("rate limited output should return error")
	}
	if !strings.Contains(err.Error(), "rate/quota limited") {
		t.Errorf("error should mention rate/quota limited: %v", err)
	}
}

func TestRunReview_ContextCanceled(t *testing.T) {
	// given
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	// when
	_, err := RunReview(ctx, "sleep 10", t.TempDir())

	// then
	if err == nil {
		t.Error("canceled context should return error")
	}
}
