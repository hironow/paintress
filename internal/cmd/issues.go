package cmd

import (
	"fmt"

	"github.com/hironow/paintress/internal/session"
	"github.com/spf13/cobra"
)

func newIssuesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues [path]",
		Short: "DEPRECATED post jun15 MCP pivot (refs/issues/0027)",
		Long: `DEPRECATED post jun15 MCP pivot.

The previous implementation invoked the Claude CLI subprocess to query
Linear issues via the Linear MCP tools. Post jun15 MCP pivot
(refs/issues/0027 + 0028 §4.2 residue cleanup), headless Claude
invocations are forbidden. Use claude code with the paintress MCP
server attached instead:

  claude --plugin-dir ./plugins/paintress \
         --mcp-config '{"paintress":{"command":"paintress","args":["mcp"]}}'

Then call the paintress.next_issue MCP tool from your session.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runIssues,
	}

	cmd.Flags().StringP("state", "s", "", "(deprecated) state filter, no longer applied")

	return cmd
}

func runIssues(cmd *cobra.Command, _ []string) error {
	fmt.Fprintln(cmd.ErrOrStderr(),
		"paintress issues: deprecated post jun15 MCP pivot (refs/issues/0027). "+
			"Use claude code with paintress MCP attached and call the paintress.next_issue tool.")
	return session.ErrMCPPivotDeprecated
}
