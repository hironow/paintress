package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
)

// newMCPCommand exposes `paintress mcp` as a stdio MCP server entry
// point for the refs/issues/0027 jun15 MCP pivot. A claude code
// interactive session loads this binary via --mcp-config and calls
// paintress tools from inside the human-initiated subscription quota.
//
// continent is resolved from the cwd (= operator launches `paintress
// mcp` from the project root). The real-impl tools (next_issue /
// update_gradient / append_journal) use it to read/write journal /
// pr-index / event-store state. ping is continent-agnostic.
func newMCPCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run paintress as an MCP server over stdio (expedition journal/gradient data plane)",
		Long: `Start a Model Context Protocol server reading JSON-RPC 2.0
messages on stdin and writing responses on stdout.

Designed for embedding in a claude code interactive session via
--mcp-config so inference stays on the session's subscription quota
rather than crossing into the Agent SDK credit pool that gates
'claude -p' from 2026-06-15.

The continent (= project root) is resolved from the current working
directory. paintress.next_issue reads pr-index.jsonl + journal/ under
this directory to surface completed issue ids + next expedition
number. The claude code session itself queries linear-mcp for raw
issue data and uses paintress.next_issue's completed_issue_ids to
exclude already-done work.

Exposes paintress.ping, paintress.next_issue (reads journal +
pr-index to surface completed issue ids + next expedition number),
and paintress.update_gradient + paintress.append_journal (persist
gradient / expedition-completed events to the event store, with a
journal/ + pr-index filesystem write).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			continent, err := os.Getwd()
			if err != nil {
				return err
			}
			// Emitter is wired by default (refs/issues/0027 Phase 4
			// follow-up #4): update_gradient / append_journal append
			// EventGradientChanged / EventExpeditionCompleted via the
			// usecase ExpeditionEventEmitter. LLM firing remains
			// human-initiated — events land only when the claude-code
			// session calls the MCP tool.
			stateDir := filepath.Join(continent, domain.StateDir)
			store := session.NewEventStore(stateDir, nil)
			emitter := usecase.NewExpeditionEventEmitter(
				cmd.Context(),
				domain.NewExpeditionAggregate(),
				store,
				nil,
				&domain.NopLogger{},
				"paintress.mcp",
			)
			srv := session.NewMCPServer(cmd.InOrStdin(), cmd.OutOrStdout(), nil).
				WithContinent(continent).
				WithEmitter(emitter)
			return srv.Serve(cmd.Context())
		},
	}
}
