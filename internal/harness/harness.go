// Package harness provides the policy and verification layer for paintress.
// It re-exports all functions from sub-packages (policy, verifier) as a
// single entry point for callers outside the harness package.
package harness

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/policy"
	"github.com/hironow/paintress/internal/harness/verifier"
)

// --- policy: gradient ---

// GradientGauge is a type alias for the policy GradientGauge.
type GradientGauge = policy.GradientGauge

// NewGradientGauge creates a new GradientGauge.
var NewGradientGauge = policy.NewGradientGauge

// ReportSeverity maps a gauge level to a D-Mail severity string.
var ReportSeverity = policy.ReportSeverity

// --- policy: reserve ---

// ReserveParty is a type alias for the policy ReserveParty.
type ReserveParty = policy.ReserveParty

// NewReserveParty creates a new ReserveParty.
var NewReserveParty = policy.NewReserveParty

// --- policy: retry ---

// RetryTracker is a type alias for the policy RetryTracker.
type RetryTracker = policy.RetryTracker

// NewRetryTracker creates a new unlimited RetryTracker.
var NewRetryTracker = policy.NewRetryTracker

// NewRetryTrackerWithMax creates a new RetryTracker with a cap.
var NewRetryTrackerWithMax = policy.NewRetryTrackerWithMax

// RetryKey returns the canonical key for an issue set.
var RetryKey = policy.RetryKey

// --- policy: dmail ---

// FormatDMailForPrompt formats d-mails for prompt injection.
var FormatDMailForPrompt = policy.FormatDMailForPrompt

// NewReportDMail creates a report D-Mail from an ExpeditionReport.
var NewReportDMail = policy.NewReportDMail

// BuildFollowUpPrompt builds a follow-up prompt for matched D-Mails.
var BuildFollowUpPrompt = policy.BuildFollowUpPrompt

// FilterHighSeverity returns only HIGH severity d-mails.
var FilterHighSeverity = policy.FilterHighSeverity

// --- policy: review ---

// HasReviewComments checks for actionable review comment indicators.
var HasReviewComments = policy.HasReviewComments

// IsRateLimited checks for rate/quota limiting signals.
var IsRateLimited = policy.IsRateLimited

// ExpandReviewCmd replaces placeholders in the review command.
var ExpandReviewCmd = policy.ExpandReviewCmd

// BuildReviewFixPrompt creates a fix prompt for review comments.
var BuildReviewFixPrompt = policy.BuildReviewFixPrompt

// SummarizeReview normalizes and truncates review output.
var SummarizeReview = policy.SummarizeReview

// --- policy: strategy ---

// FixStrategy identifies the approach for a review-fix cycle.
type FixStrategy = policy.FixStrategy

// Strategy constants.
const (
	StrategyDirect    = policy.StrategyDirect
	StrategyDecompose = policy.StrategyDecompose
	StrategyRewrite   = policy.StrategyRewrite
)

// StrategyForCycle returns the fix strategy for a given cycle number.
var StrategyForCycle = policy.StrategyForCycle

// BuildReviewFixPromptWithStrategy creates a fix prompt with strategy hint.
var BuildReviewFixPromptWithStrategy = policy.BuildReviewFixPromptWithStrategy

// --- policy: reflection ---

// ReflectionAccumulator is a type alias for the policy ReflectionAccumulator.
type ReflectionAccumulator = policy.ReflectionAccumulator

// ReflectionCycle is a type alias for the policy ReflectionCycle.
type ReflectionCycle = policy.ReflectionCycle

// NewReflectionAccumulator creates an empty ReflectionAccumulator.
var NewReflectionAccumulator = policy.NewReflectionAccumulator

// BuildReviewFixPromptWithReflection creates a fix prompt with reflection.
var BuildReviewFixPromptWithReflection = policy.BuildReviewFixPromptWithReflection

// --- policy: stagnation ---

// CountPriorityTags counts priority tags in review output.
var CountPriorityTags = policy.CountPriorityTags

// IsStagnant checks if the review loop has stagnated.
var IsStagnant = policy.IsStagnant

// --- policy: wave ---

// ProjectWaveState builds wave progress from D-Mails.
var ProjectWaveState = policy.ProjectWaveState

// ExpeditionTargetsFromWaves converts pending wave steps into targets.
var ExpeditionTargetsFromWaves = policy.ExpeditionTargetsFromWaves

// --- policy: lumina ---

// FormatLuminaForPrompt formats Luminas for prompt injection.
var FormatLuminaForPrompt = policy.FormatLuminaForPrompt

// --- verifier ---

// ValidateDMail checks that a DMail conforms to D-Mail schema v1.
var ValidateDMail = verifier.ValidateDMail

// ClassifyProviderError classifies stderr output by provider.
func ClassifyProviderError(provider domain.Provider, stderr string) domain.ProviderErrorInfo {
	return verifier.ClassifyProviderError(provider, stderr)
}
