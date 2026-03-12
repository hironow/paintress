package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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

func (p *Paintress) runReviewLoop(ctx context.Context, report *domain.ExpeditionReport, budget time.Duration, workDir string) domain.ReviewGateStatus {
	ctx, loopSpan := platform.Tracer.Start(ctx, "review.loop",
		trace.WithAttributes(
			attribute.String("pr_url", platform.SanitizeUTF8(report.PRUrl)),
			attribute.String("branch", platform.SanitizeUTF8(report.Branch)),
		),
	)
	defer loopSpan.End()

	reviewDir := workDir
	if reviewDir == "" {
		reviewDir = p.config.Continent
	}

	var consumed time.Duration

	notResolved := func(cycle int, comments string) domain.ReviewGateStatus {
		return domain.ReviewGateStatus{Cycle: cycle, MaxCycles: maxReviewGateCycles, LastComments: comments}
	}

	reviewTimeout := max(
		time.Duration(p.config.TimeoutSec)*time.Second/time.Duration(maxReviewGateCycles),
		minReviewTimeout,
	)
	var lastComments string
	for cycle := 1; cycle <= maxReviewGateCycles; cycle++ {
		if ctx.Err() != nil {
			if lastComments != "" {
				if report.Insight != "" {
					report.Insight += " | "
				}
				report.Insight += "Review interrupted: " + domain.SummarizeReview(lastComments)
			}
			p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("review_error"), ctx.Err()))
			return notResolved(cycle, lastComments)
		}

		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("review_running"), cycle, maxReviewGateCycles))

		_, revSpan := platform.Tracer.Start(ctx, "review.command", // nosemgrep: adr0003-otel-span-without-defer-end -- End() called per branch [permanent]
			trace.WithAttributes(attribute.Int("cycle", cycle)),
		)
		reviewCtx, reviewCancel := context.WithTimeout(ctx, reviewTimeout)
		expandedCmd := domain.ExpandReviewCmd(p.config.ReviewCmd, reviewDir, report.Branch)
		result, err := RunReview(reviewCtx, expandedCmd, reviewDir)
		reviewCancel()
		if err != nil {
			revSpan.End()
			if lastComments != "" {
				if report.Insight != "" {
					report.Insight += " | "
				}
				report.Insight += "Review interrupted: " + domain.SummarizeReview(lastComments)
			}
			p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("review_error"), err))
			return notResolved(cycle, lastComments)
		}

		revSpan.SetAttributes(attribute.Bool("passed", result.Passed))
		revSpan.End()

		if result.Passed {
			p.Logger.OK("%s", domain.Msg("review_passed"))
			return domain.ReviewGateStatus{Passed: true, Cycle: cycle, MaxCycles: maxReviewGateCycles}
		}

		lastComments = result.Comments
		p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("review_comments"), cycle))

		branch := strings.TrimSpace(report.Branch)
		if branch == "" || strings.EqualFold(branch, "none") {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Reviewfix skipped: no valid branch"
			return notResolved(cycle, lastComments)
		}

		gitCtx, gitCancel := context.WithTimeout(ctx, gitCmdTimeout)
		gitCmd := exec.CommandContext(gitCtx, "git", "checkout", branch)
		gitCmd.Dir = reviewDir
		err = gitCmd.Run()
		gitCancel()
		if err != nil {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += fmt.Sprintf("Reviewfix skipped: checkout %s failed: %v", branch, err)
			return notResolved(cycle, lastComments)
		}

		remaining := budget - consumed
		if remaining <= 0 {
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Review not fully resolved: " + domain.SummarizeReview(lastComments)
			p.Logger.Warn("%s", domain.Msg("review_limit"))
			return notResolved(cycle, lastComments)
		}

		fixCtx, fixCancel := context.WithTimeout(ctx, remaining)

		prompt := domain.BuildReviewFixPrompt(branch, result.Comments)

		model := p.reserve.ActiveModel()
		_, fixSpan := platform.Tracer.Start(fixCtx, "reviewfix.claude", // nosemgrep: adr0003-otel-span-without-defer-end -- End() called after Run [permanent]
			trace.WithAttributes(
				append([]attribute.KeyValue{
					attribute.Int("cycle", cycle),
					attribute.String("model", platform.SanitizeUTF8(model)),
				}, platform.GenAISpanAttrs(model)...)...,
			),
		)

		p.Logger.Info("%s", fmt.Sprintf(domain.Msg("reviewfix_running"), model))
		start := time.Now()
		out, err := p.claude.Run(fixCtx, prompt, io.Discard,
			port.WithContinue(),
			port.WithWorkDir(reviewDir),
			port.WithAllowedTools(ReviewFixAllowedTools...),
			port.WithModel(model),
		)
		consumed += time.Since(start)
		fixSpan.End()
		fixCancel()

		if err != nil {
			p.Logger.Warn("%s", fmt.Sprintf(domain.Msg("reviewfix_error"), err))
			if report.Insight != "" {
				report.Insight += " | "
			}
			report.Insight += "Reviewfix failed: " + domain.SummarizeReview(out)
			return notResolved(cycle, lastComments)
		}
	}

	if report.Insight != "" {
		report.Insight += " | "
	}
	report.Insight += "Review not fully resolved: " + domain.SummarizeReview(lastComments)
	p.Logger.Warn("%s", domain.Msg("review_limit"))
	return notResolved(maxReviewGateCycles, lastComments)
}

// runSkipReview runs review_cmd against past PRs from pr-index.jsonl when an
// expedition is skipped. This prevents idle loops when all issues are In Review.
func (p *Paintress) runSkipReview(ctx context.Context, workDir string, expStart time.Time) {
	if ctx.Err() != nil {
		return
	}
	entries, err := ReadPRIndex(p.config.Continent)
	if err != nil {
		p.Logger.Warn("skip review: read pr index: %v", err)
		return
	}
	if len(entries) == 0 {
		return
	}

	reviewDir := workDir
	if reviewDir == "" {
		reviewDir = p.config.Continent
	}

	totalTimeout := time.Duration(p.config.TimeoutSec) * time.Second
	remaining := totalTimeout - time.Since(expStart)
	if remaining <= 0 {
		return
	}

	// Review the most recent PR (last entry in append-only index).
	latest := entries[len(entries)-1]
	p.Logger.Info("Skip re-review: checking past PR %s (%s)", latest.PRUrl, latest.IssueID)

	// Build a synthetic report for the review loop.
	syntheticReport := &domain.ExpeditionReport{
		IssueID: latest.IssueID,
		PRUrl:   latest.PRUrl,
		Branch:  "", // branch unknown from index; review runs in current dir
	}

	reviewStatus := p.runReviewLoop(ctx, syntheticReport, remaining, reviewDir)
	if reviewStatus.Cycle > 0 {
		// Review was executed (pass or comments found) — productive work, not idle.
		// Reset consecutive skip counter to prevent premature termination.
		p.consecutiveSkips.Store(0)
		if reviewStatus.Passed {
			p.Logger.OK("Skip re-review: past PR %s passed review", latest.PRUrl)
		} else {
			p.Logger.Info("Skip re-review: past PR %s has unresolved comments", latest.PRUrl)
		}
	}
}

func (p *Paintress) runFollowUp(ctx context.Context, dmails []domain.DMail, workDir string, remaining time.Duration) {
	if len(dmails) == 0 {
		return
	}
	if ctx.Err() != nil {
		return
	}
	if remaining <= 0 {
		p.Logger.Warn("Follow-up skipped: no remaining time budget")
		return
	}

	prompt := domain.BuildFollowUpPrompt(dmails)

	model := p.reserve.ActiveModel()
	_, followUpSpan := platform.Tracer.Start(ctx, "followup.claude",
		trace.WithAttributes(
			append([]attribute.KeyValue{
				attribute.String("model", platform.SanitizeUTF8(model)),
				attribute.Int("matched_dmails", len(dmails)),
			}, platform.GenAISpanAttrs(model)...)...,
		),
	)
	defer followUpSpan.End()

	p.Logger.Info("Follow-up: delivering %d matched D-Mail(s) via --continue", len(dmails))

	timeout := time.Duration(p.config.TimeoutSec) * time.Second
	if remaining < timeout {
		timeout = remaining
	}
	followCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	dir := workDir
	if dir == "" {
		dir = p.config.Continent
	}

	out, err := p.claude.Run(followCtx, prompt, io.Discard,
		port.WithContinue(),
		port.WithWorkDir(dir),
		port.WithAllowedTools(ReviewFixAllowedTools...),
		port.WithModel(model),
	)
	if err != nil {
		p.Logger.Warn("Follow-up failed: %v", err)
		followUpSpan.AddEvent("followup.error",
			trace.WithAttributes(attribute.String("error", platform.SanitizeUTF8(err.Error()))),
		)
		return
	}
	p.Logger.OK("Follow-up completed (%d bytes output)", len(out))
}
