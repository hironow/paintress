package filter

import (
	"fmt"
	"strings"
)

// BuildReviewFixPrompt creates a focused prompt for fixing review comments.
func BuildReviewFixPrompt(branch string, comments string) string {
	return fmt.Sprintf(`You are on branch %s with an open PR. A code review found the following issues:

%s

Fix all review comments above. Commit and push your changes. Do not create a new branch or PR.
Keep fixes focused — only address the review comments, do not refactor unrelated code.
If a review comment is unclear or you cannot resolve it after a reasonable attempt, skip it and move on to the next.
Do not change the Linear issue status — it stays in its current state until the next Expedition.`, branch, comments)
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
