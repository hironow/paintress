package filter

import (
	"fmt"

	"github.com/hironow/paintress/internal/harness/policy"
)

// strategyHint returns the additional prompt hint for a non-Direct strategy.
func strategyHint(strategy policy.FixStrategy) string {
	switch strategy {
	case policy.StrategyDecompose:
		return "\nStrategy hint: decompose the review comments into small, independent steps. Fix each step separately before moving to the next."
	case policy.StrategyRewrite:
		return "\nStrategy hint: if a fix is not straightforward, rewrite the affected section from scratch to address all comments cleanly."
	default:
		return ""
	}
}

// BuildReviewFixPromptWithStrategy creates a fix prompt with a cycle-specific strategy hint.
func BuildReviewFixPromptWithStrategy(branch string, comments string, strategy policy.FixStrategy) string {
	hint := strategyHint(strategy)
	return fmt.Sprintf(`You are on branch %s with an open PR. A code review found the following issues:

%s

Fix all review comments above. Commit and push your changes. Do not create a new branch or PR.
Keep fixes focused — only address the review comments, do not refactor unrelated code.
If a review comment is unclear or you cannot resolve it after a reasonable attempt, skip it and move on to the next.
Do not change the Linear issue status — it stays in its current state until the next Expedition.%s`, branch, comments, hint)
}
