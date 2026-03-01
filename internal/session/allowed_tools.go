package session

// ReviewFixAllowedTools restricts Claude's tool set during review-fix and
// follow-up phases. These phases only need code editing and git operations;
// build/test execution is not required.
var ReviewFixAllowedTools = []string{
	"Read", "Write", "Edit",
	"Bash(git:*)", "Bash(gh:*)",
	"Bash(ls:*)", "Bash(grep:*)", "Bash(cat:*)",
	"Bash(head:*)", "Bash(sed:*)", "Bash(awk:*)",
}
