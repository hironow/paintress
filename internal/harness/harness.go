// Package harness provides the policy and verification layer for paintress.
// It re-exports all functions from sub-packages (policy, verifier, filter) as a
// single entry point for callers outside the harness package.
package harness

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/filter"
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

// NewReportDMail creates a report D-Mail from an ExpeditionReport.
var NewReportDMail = policy.NewReportDMail

// FilterHighSeverity returns only HIGH severity d-mails.
var FilterHighSeverity = policy.FilterHighSeverity

// --- policy: review ---

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

// --- policy: reflection ---

// ReflectionAccumulator is a type alias for the policy ReflectionAccumulator.
type ReflectionAccumulator = policy.ReflectionAccumulator

// ReflectionCycle is a type alias for the policy ReflectionCycle.
type ReflectionCycle = policy.ReflectionCycle

// NewReflectionAccumulator creates an empty ReflectionAccumulator.
var NewReflectionAccumulator = policy.NewReflectionAccumulator

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

// --- filter: prompt registry ---

// PromptRegistry is a type alias for the filter PromptRegistry.
type PromptRegistry = filter.PromptRegistry

// PromptEntry is a type alias for the filter PromptEntry.
type PromptEntry = filter.PromptEntry

// NewPromptRegistry creates a new PromptRegistry from embedded YAML files.
var NewPromptRegistry = filter.NewRegistry

// DefaultPromptRegistry returns the package-level PromptRegistry singleton.
var DefaultPromptRegistry = filter.Default

// MustDefaultPromptRegistry returns the singleton or panics.
var MustDefaultPromptRegistry = filter.MustDefault

// --- filter: dmail ---

// FormatDMailForPrompt formats d-mails for prompt injection.
var FormatDMailForPrompt = filter.FormatDMailForPrompt

// BuildFollowUpPrompt builds a follow-up prompt for matched D-Mails.
var BuildFollowUpPrompt = filter.BuildFollowUpPrompt

// --- filter: lumina ---

// FormatLuminaForPrompt formats Luminas for prompt injection.
var FormatLuminaForPrompt = filter.FormatLuminaForPrompt

// --- filter: review ---

// BuildReviewFixPrompt creates a fix prompt for review comments.
var BuildReviewFixPrompt = filter.BuildReviewFixPrompt

// ExpandReviewCmd replaces placeholders in the review command.
var ExpandReviewCmd = filter.ExpandReviewCmd

// BuildReviewFixPromptWithStrategy creates a fix prompt with strategy hint.
var BuildReviewFixPromptWithStrategy = filter.BuildReviewFixPromptWithStrategy

// BuildReviewFixPromptWithReflection creates a fix prompt with reflection.
var BuildReviewFixPromptWithReflection = filter.BuildReviewFixPromptWithReflection

// --- verifier ---

// HasReviewComments checks for actionable review comment indicators.
var HasReviewComments = verifier.HasReviewComments

// IsRateLimited checks for rate/quota limiting signals.
var IsRateLimited = verifier.IsRateLimited

// ValidateDMail checks that a DMail conforms to D-Mail schema v1.
var ValidateDMail = verifier.ValidateDMail

// ClassifyProviderError classifies stderr output by provider.
func ClassifyProviderError(provider domain.Provider, stderr string) domain.ProviderErrorInfo {
	return verifier.ClassifyProviderError(provider, stderr)
}
