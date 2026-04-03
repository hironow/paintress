package policy

import "fmt"

// FixStrategy identifies the approach for a review-fix cycle.
type FixStrategy string

const (
	// StrategyDirect applies review comments directly without additional guidance.
	StrategyDirect FixStrategy = "direct"
	// StrategyDecompose breaks review comments into smaller sub-tasks before fixing.
	StrategyDecompose FixStrategy = "decompose"
	// StrategyRewrite rewrites the affected section from scratch to resolve comments.
	StrategyRewrite FixStrategy = "rewrite"
)

// strategies is the ordered rotation of fix strategies.
var strategies = []FixStrategy{
	StrategyDirect,
	StrategyDecompose,
	StrategyRewrite,
}

// StrategyForCycle returns the fix strategy for the given cycle number.
// Cycles rotate through Direct → Decompose → Rewrite and repeat.
// Cycle numbering starts at 1.
func StrategyForCycle(cycle int) FixStrategy {
	if cycle < 1 {
		cycle = 1
	}
	idx := (cycle - 1) % len(strategies)
	return strategies[idx]
}

// strategyHint returns the additional prompt hint for a non-Direct strategy.
func strategyHint(strategy FixStrategy) string {
	switch strategy {
	case StrategyDecompose:
		return "\nStrategy hint: decompose the review comments into small, independent steps. Fix each step separately before moving to the next."
	case StrategyRewrite:
		return "\nStrategy hint: if a fix is not straightforward, rewrite the affected section from scratch to address all comments cleanly."
	default:
		return ""
	}
}

// BuildReviewFixPromptWithStrategy creates a fix prompt with a cycle-specific strategy hint.
func BuildReviewFixPromptWithStrategy(branch string, comments string, strategy FixStrategy) string {
	hint := strategyHint(strategy)
	return fmt.Sprintf(`You are on branch %s with an open PR. A code review found the following issues:

%s

Fix all review comments above. Commit and push your changes. Do not create a new branch or PR.
Keep fixes focused — only address the review comments, do not refactor unrelated code.
If a review comment is unclear or you cannot resolve it after a reasonable attempt, skip it and move on to the next.
Do not change the Linear issue status — it stays in its current state until the next Expedition.%s`, branch, comments, hint)
}
