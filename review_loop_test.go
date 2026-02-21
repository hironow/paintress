package paintress

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeScript creates an executable shell script.
func writeScript(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/bash\n"+content), 0755); err != nil {
		t.Fatalf("write script %s: %v", path, err)
	}
}

// newTestPaintress creates a minimal Paintress for review loop tests.
func newTestPaintress(t *testing.T, dir string, timeoutSec int, reviewCmd string, claudeCmd string) *Paintress {
	t.Helper()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run", "logs"), 0755)

	cfg := Config{
		Continent:  dir,
		TimeoutSec: timeoutSec,
		ClaudeCmd:  claudeCmd,
		ReviewCmd:  reviewCmd,
		BaseBranch: "main",
		Model:      "opus",
	}
	return NewPaintress(cfg, NewLogger(io.Discard, false), io.Discard, nil)
}

// TestReviewLoop_ReviewTimeDoesNotConsumeBudget verifies that slow review
// commands do not eat into the fix budget. With budget=2s and review sleeping
// 0.5s per cycle, if review time were counted the budget would be exhausted.
// Since review time is NOT counted and fix is instant, all 3 cycles complete.
func TestReviewLoop_ReviewTimeDoesNotConsumeBudget(t *testing.T) {
	// given
	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, "sleep 0.5\necho '[P2] Fix something'\n")

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "exit 0\n") // instant fix

	p := newTestPaintress(t, dir, 30, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	// when — pass 2s budget (simulating expedition consumed most of the 30s)
	start := time.Now()
	p.runReviewLoop(context.Background(), report, 2*time.Second, "")
	elapsed := time.Since(start)

	// then — all 3 review cycles should have run (~1.5s review + ~0s fix)
	if elapsed < 1400*time.Millisecond {
		t.Errorf("expected all 3 review cycles (~1.5s), completed in %v", elapsed)
	}
	if !strings.Contains(report.Insight, "Review not fully resolved") {
		t.Errorf("expected all cycles exhausted, got insight: %q", report.Insight)
	}
}

// TestReviewLoop_FixTimeConsumesBudget verifies that fix phase execution time
// is correctly tracked against the budget. With budget=1s and each fix taking
// 0.4s, the budget is exhausted within 3 cycles.
func TestReviewLoop_FixTimeConsumesBudget(t *testing.T) {
	// given
	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, "echo '[P2] Fix something'\n")

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "sleep 0.4\nexit 0\n")

	p := newTestPaintress(t, dir, 30, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	// when — pass 1s budget
	p.runReviewLoop(context.Background(), report, 1*time.Second, "")

	// then — budget exhaustion should stop the loop
	if report.Insight == "" {
		t.Error("expected insight about review not fully resolved or fix timeout")
	}
}

// TestReviewLoop_GitCheckoutTimeoutPreventsHang verifies that
// a hanging git checkout does not block the review loop indefinitely.
func TestReviewLoop_GitCheckoutTimeoutPreventsHang(t *testing.T) {
	// given — override the git timeout to 2s
	old := gitCmdTimeout
	gitCmdTimeout = 2 * time.Second
	defer func() { gitCmdTimeout = old }()

	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, "echo '[P2] Fix something'\n")

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "exit 0\n")

	fakeGitDir := filepath.Join(dir, "fakebin")
	os.MkdirAll(fakeGitDir, 0755)
	writeScript(t, filepath.Join(fakeGitDir, "git"), "sleep 999\n")

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", fakeGitDir+":"+origPath)
	defer os.Setenv("PATH", origPath)

	p := newTestPaintress(t, dir, 30, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	// when
	start := time.Now()
	p.runReviewLoop(context.Background(), report, 30*time.Second, "")
	elapsed := time.Since(start)

	// then
	if elapsed > 5*time.Second {
		t.Errorf("git checkout should timeout around 2s, took %v", elapsed)
	}
	if !strings.Contains(report.Insight, "checkout") {
		t.Errorf("insight should mention checkout failure, got: %q", report.Insight)
	}
}

// TestReviewLoop_ParentContextCancellationStopsLoop verifies that
// cancelling the parent context stops the review loop promptly.
func TestReviewLoop_ParentContextCancellationStopsLoop(t *testing.T) {
	// given
	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, "sleep 5\necho '[P2] Fix something'\n")

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "exit 0\n")

	p := newTestPaintress(t, dir, 30, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	ctx, cancel := context.WithCancel(context.Background())

	// when — cancel after 500ms
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	p.runReviewLoop(ctx, report, 30*time.Second, "")
	elapsed := time.Since(start)

	// then
	if elapsed > 3*time.Second {
		t.Errorf("parent cancellation should stop loop quickly, took %v", elapsed)
	}
}

// TestReviewLoop_ReviewCmdTimeoutDerivedFromConfig verifies that each
// review command call is bounded by TimeoutSec / maxReviewCycles.
// With TimeoutSec=6 and maxReviewCycles=3, each review gets 2s.
// minReviewTimeout is lowered so the computed 2s is not clamped.
func TestReviewLoop_ReviewCmdTimeoutDerivedFromConfig(t *testing.T) {
	// given — lower clamp so computed timeout (2s) is used
	old := minReviewTimeout
	minReviewTimeout = 1 * time.Second
	defer func() { minReviewTimeout = old }()

	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, "sleep 999\necho '[P2] Fix something'\n") // hangs

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "exit 0\n")

	// TimeoutSec=6 → review timeout = 6s/3 = 2s per call
	p := newTestPaintress(t, dir, 6, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	// when
	start := time.Now()
	p.runReviewLoop(context.Background(), report, 30*time.Second, "")
	elapsed := time.Since(start)

	// then — review should timeout at ~2s (6s/3), not hang
	if elapsed > 5*time.Second {
		t.Errorf("review should timeout at ~2s (TimeoutSec/3), took %v", elapsed)
	}
}

// TestReviewLoop_BudgetSharedWithExpedition verifies that a small remaining
// budget (simulating expedition consumed most of the total timeout) correctly
// limits the number of fix cycles.
func TestReviewLoop_BudgetSharedWithExpedition(t *testing.T) {
	// given — budget=0.5s, fix takes 0.3s each → only 1 fix can run
	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, "echo '[P2] Fix something'\n")

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "sleep 0.3\nexit 0\n")

	p := newTestPaintress(t, dir, 30, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	// when — only 0.5s budget remaining (expedition consumed 29.5s of 30s)
	start := time.Now()
	p.runReviewLoop(context.Background(), report, 500*time.Millisecond, "")
	elapsed := time.Since(start)

	// then — should complete quickly (only 1-2 fix cycles possible)
	if elapsed > 3*time.Second {
		t.Errorf("small budget should limit fix cycles, took %v", elapsed)
	}
	if report.Insight == "" {
		t.Error("expected insight about budget exhaustion or fix timeout")
	}
}

// TestReviewLoop_ShortTimeoutStillRunsReview verifies that a very short
// --timeout value does not produce a zero or near-zero review timeout
// that would cancel RunReview instantly. The review timeout should be
// clamped to a minimum so the review command has a chance to execute.
func TestReviewLoop_ShortTimeoutStillRunsReview(t *testing.T) {
	// given — TimeoutSec=0 → naive reviewTimeout = 0s/3 = 0 → instant cancel
	// The clamp should ensure the review command still gets a chance to execute.
	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	marker := filepath.Join(dir, ".review-ran")
	reviewScript := filepath.Join(dir, "review.sh")
	writeScript(t, reviewScript, fmt.Sprintf("touch \"%s\"\necho 'all good'\n", marker))

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "exit 0\n")

	// TimeoutSec=0 → without clamp, reviewTimeout=0 → instant cancel
	p := newTestPaintress(t, dir, 0, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	// when
	p.runReviewLoop(context.Background(), report, 30*time.Second, "")

	// then — the review script should have actually executed
	if _, err := os.Stat(marker); os.IsNotExist(err) {
		t.Error("review command was never executed; reviewTimeout clamp needed")
	}
}

// TestReviewLoop_ReviewErrorPreservesLastComments verifies that when
// the review command fails on a later cycle, the review comments found
// in earlier cycles are still recorded in the report insight.
func TestReviewLoop_ReviewErrorPreservesLastComments(t *testing.T) {
	// given — cycle 1: review finds comments, fix succeeds
	//         cycle 2: review command fails → lastComments should be preserved
	dir := t.TempDir()
	setupGitRepoWithBranch(t, dir, "feat/test")

	reviewScript := filepath.Join(dir, "review.sh")
	markerFile := filepath.Join(dir, ".review-done")
	writeScript(t, reviewScript, fmt.Sprintf(`if [ ! -f "%s" ]; then
    touch "%s"
    echo '[P2] Fix something'
else
    echo 'internal error'
    exit 1
fi
`, markerFile, markerFile))

	fakeClaudeCmd := filepath.Join(dir, "fakeclaude.sh")
	writeScript(t, fakeClaudeCmd, "exit 0\n")

	p := newTestPaintress(t, dir, 30, reviewScript, fakeClaudeCmd)
	report := &ExpeditionReport{Branch: "feat/test"}

	// when
	p.runReviewLoop(context.Background(), report, 30*time.Second, "")

	// then — cycle 1's review comments should be preserved in insight
	if report.Insight == "" {
		t.Error("expected insight preserving review comments from earlier cycle")
	}
	if !strings.Contains(report.Insight, "Review") {
		t.Errorf("insight should mention review comments, got: %q", report.Insight)
	}
}
