package cmd

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/hironow/paintress/internal/domain"
)

// newRunCommand is a deprecation stub. The autonomous expedition loop
// drove a headless 'claude -p' invocation, removed in the jun15 MCP
// pivot. Expeditions now run from a claude code interactive session via
// the /expedition-next skill, which drives paintress's MCP tools.
func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [path]",
		Short: "Deprecated (jun15 MCP pivot): use claude code + /expedition-next skill",
		Long: `Deprecated by the jun15 MCP pivot (2026-06-15 credit-pool split).

The autonomous expedition loop no longer drives a headless 'claude -p'
invocation. Run expeditions from a claude code interactive session via
the /expedition-next skill, which drives paintress's MCP tools
(next_issue / update_gradient / append_journal). Start the data-plane
server with 'paintress mcp'.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("paintress run is deprecated (jun15 MCP pivot): drive expeditions from a claude code session via the /expedition-next skill + paintress MCP tools (next_issue / update_gradient / append_journal); see 'paintress mcp'")
		},
	}

	cmd.Flags().Int("max-expeditions", 50, "Maximum number of expeditions")
	cmd.Flags().IntP("timeout", "t", 1980, "Timeout per expedition in seconds (default: 33min)")
	cmd.Flags().StringP("model", "m", "opus", "Model(s) comma-separated for reserve: opus,sonnet,haiku")
	cmd.Flags().StringP("base-branch", "b", "main", "Base branch")
	cmd.Flags().String("claude-cmd", domain.DefaultClaudeCmd, "Claude Code CLI command name")
	cmd.Flags().String("dev-cmd", "npm run dev", "Dev server command")
	cmd.Flags().String("dev-dir", "", "Dev server working directory (defaults to repo path)")
	cmd.Flags().String("dev-url", "http://localhost:3000", "Dev server URL")
	cmd.Flags().String("review-cmd", "", "Code review command after PR creation")
	cmd.Flags().IntP("workers", "w", 1, "Number of worktrees in pool (0 = direct execution)")
	cmd.Flags().String("setup-cmd", "", "Command to run after worktree creation (e.g. 'bun install')")
	cmd.Flags().Bool("no-dev", false, "Skip dev server startup")
	cmd.Flags().BoolP("dry-run", "n", false, "Generate prompts only")
	cmd.Flags().String("notify-cmd", "", "Notification command ({title}, {message} placeholders)")
	cmd.Flags().String("approve-cmd", "", "Approval command ({message} placeholder, exit 0 = approve)")
	cmd.Flags().Bool("auto-approve", false, "Skip approval gate for HIGH severity D-Mail")
	cmd.Flags().Duration("idle-timeout", domain.DefaultIdleTimeout, "D-Mail waiting phase timeout (0 = 24h safety cap, negative = disable waiting)")

	return cmd
}
