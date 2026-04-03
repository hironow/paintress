package filter

import (
	"fmt"

	"github.com/hironow/paintress/internal/harness/policy"
)

// BuildReviewFixPromptWithReflection creates a fix prompt that includes reflection history
// from a ReflectionAccumulator to help Claude avoid repeating prior failures.
func BuildReviewFixPromptWithReflection(branch string, comments string, acc *policy.ReflectionAccumulator) string {
	reflection := acc.FormatForPrompt()
	if reflection == "" {
		return BuildReviewFixPrompt(branch, comments)
	}
	return fmt.Sprintf(`You are on branch %s with an open PR. A code review found the following issues:

%s

Fix all review comments above. Commit and push your changes. Do not create a new branch or PR.
Keep fixes focused — only address the review comments, do not refactor unrelated code.
If a review comment is unclear or you cannot resolve it after a reasonable attempt, skip it and move on to the next.
Do not change the Linear issue status — it stays in its current state until the next Expedition.

---

%s`, branch, comments, reflection)
}
