package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/hironow/paintress/internal/session"
	"github.com/spf13/cobra"
)

func newArchivePruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive-prune <repo-path>",
		Short: "Prune old archived d-mails",
		Long: `Prune archived d-mail files older than a specified number of days.

By default runs in dry-run mode, listing candidates without deleting.
Use --execute to perform actual deletion. The archive/ directory is
git-tracked, so deletions should be reviewed and committed.`,
		Example: `  # Dry run: list files older than 30 days
  paintress archive-prune /path/to/repo

  # Delete files older than 14 days
  paintress archive-prune --days 14 --execute /path/to/repo

  # JSON output for scripting
  paintress archive-prune -o json /path/to/repo`,
		Args: cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			if days <= 0 {
				return fmt.Errorf("--days must be positive, got %d", days)
			}
			return nil
		},
		RunE: runArchivePrune,
	}

	cmd.Flags().IntP("days", "d", 30, "Number of days threshold")
	cmd.Flags().BoolP("execute", "x", false, "Execute deletion (dry-run by default)")

	return cmd
}

func runArchivePrune(cmd *cobra.Command, args []string) error {
	repoPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	days, _ := cmd.Flags().GetInt("days")
	execute, _ := cmd.Flags().GetBool("execute")
	outputFmt, _ := cmd.Flags().GetString("output")
	stateDir := filepath.Join(repoPath, ".expedition")

	// Collect archive candidates (dry-run to list only).
	archiveResult, err := session.ArchivePrune(repoPath, days, false)
	if err != nil {
		return err
	}

	// Collect event file candidates.
	eventFiles, eventErr := session.ListExpiredEventFiles(stateDir, days)
	if eventErr != nil {
		return fmt.Errorf("failed to list expired events: %w", eventErr)
	}

	w := cmd.OutOrStdout()
	ew := cmd.ErrOrStderr()

	totalCandidates := len(archiveResult.Candidates) + len(eventFiles)

	if outputFmt == "json" {
		out := struct {
			Candidates      int      `json:"candidates"`
			Deleted         int      `json:"deleted"`
			Files           []string `json:"files"`
			EventCandidates int      `json:"event_candidates"`
			EventDeleted    int      `json:"event_deleted"`
			EventFiles      []string `json:"event_files"`
		}{
			Candidates:      len(archiveResult.Candidates),
			Files:           archiveResult.Candidates,
			EventCandidates: len(eventFiles),
			EventFiles:      eventFiles,
		}
		if execute {
			execResult, execErr := session.ArchivePrune(repoPath, days, true)
			if execErr != nil {
				return execErr
			}
			out.Deleted = execResult.Deleted

			if len(eventFiles) > 0 {
				deleted, delErr := session.PruneEventFiles(stateDir, eventFiles)
				if delErr != nil {
					return fmt.Errorf("event prune failed: %w", delErr)
				}
				out.EventDeleted = len(deleted)
			}
		}
		data, jsonErr := json.Marshal(out)
		if jsonErr != nil {
			return jsonErr
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	// text output
	if totalCandidates == 0 {
		fmt.Fprintln(w, "No files older than", days, "days.")
		return nil
	}

	if len(archiveResult.Candidates) > 0 {
		fmt.Fprintf(ew, "Expired archive files (%d):\n", len(archiveResult.Candidates))
		for _, f := range archiveResult.Candidates {
			fmt.Fprintln(ew, "  "+f)
		}
	}
	if len(eventFiles) > 0 {
		fmt.Fprintf(ew, "Expired event files (%d):\n", len(eventFiles))
		for _, f := range eventFiles {
			fmt.Fprintln(ew, "  "+f)
		}
	}
	fmt.Fprintf(ew, "%d file(s) older than %d days.\n", totalCandidates, days)

	if !execute {
		fmt.Fprintln(ew, "(dry-run — pass --execute to delete)")
		return nil
	}

	// Execute: archive deletion
	if len(archiveResult.Candidates) > 0 {
		execResult, execErr := session.ArchivePrune(repoPath, days, true)
		if execErr != nil {
			return execErr
		}
		fmt.Fprintf(ew, "Pruned %d archive file(s).\n", execResult.Deleted)
	}

	// Execute: event file deletion
	if len(eventFiles) > 0 {
		deleted, delErr := session.PruneEventFiles(stateDir, eventFiles)
		if delErr != nil {
			return fmt.Errorf("event prune failed: %w", delErr)
		}
		fmt.Fprintf(ew, "Pruned %d event file(s).\n", len(deleted))
	}

	// Prune flushed outbox DB rows + incremental vacuum.
	if pruned, pruneErr := session.PruneFlushedOutbox(repoPath); pruneErr == nil && pruned > 0 {
		fmt.Fprintf(ew, "Pruned %d flushed outbox row(s).\n", pruned)
	}

	if len(archiveResult.Candidates) > 0 {
		fmt.Fprintln(ew, "Note: archive/ is git-tracked. Run 'git status' to review and commit deletions.")
	}

	return nil
}
