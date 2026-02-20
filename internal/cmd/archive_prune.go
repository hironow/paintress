package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/hironow/paintress"
	"github.com/spf13/cobra"
)

func newArchivePruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive-prune <repo-path>",
		Short: "Prune old archived d-mails",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			if days <= 0 {
				return fmt.Errorf("--days must be positive, got %d", days)
			}
			return nil
		},
		RunE: runArchivePrune,
	}

	cmd.Flags().Int("days", 30, "Number of days threshold")
	cmd.Flags().Bool("execute", false, "Execute deletion (dry-run by default)")

	return cmd
}

func runArchivePrune(cmd *cobra.Command, args []string) error {
	repoPath := args[0]
	days, _ := cmd.Flags().GetInt("days")
	execute, _ := cmd.Flags().GetBool("execute")
	outputFmt, _ := cmd.Flags().GetString("output")

	result, err := paintress.ArchivePrune(repoPath, days, execute)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	ew := cmd.ErrOrStderr()

	if outputFmt == "json" {
		out := struct {
			Candidates int      `json:"candidates"`
			Deleted    int      `json:"deleted"`
			Files      []string `json:"files"`
		}{
			Candidates: len(result.Candidates),
			Deleted:    result.Deleted,
			Files:      result.Candidates,
		}
		data, err := json.Marshal(out)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(data))
		if execute && result.Deleted < len(result.Candidates) {
			return fmt.Errorf("%d file(s) could not be deleted", len(result.Candidates)-result.Deleted)
		}
		return nil
	}

	// text output
	if len(result.Candidates) == 0 {
		fmt.Fprintln(w, "No files older than", days, "days.")
		return nil
	}

	if execute {
		fmt.Fprintf(w, "Deleted %d file(s):\n", result.Deleted)
	} else {
		fmt.Fprintf(w, "Files older than %d days (%d file(s), dry-run):\n", days, len(result.Candidates))
	}
	for _, f := range result.Candidates {
		fmt.Fprintln(w, "  "+f)
	}
	if !execute {
		fmt.Fprintln(w, "\nRun with --execute to delete.")
	}
	fmt.Fprintln(ew, "Note: archive/ is git-tracked. Run 'git status' to review and commit deletions.")

	if execute && result.Deleted < len(result.Candidates) {
		failed := len(result.Candidates) - result.Deleted
		return fmt.Errorf("%d file(s) could not be deleted (permission denied or locked)", failed)
	}
	return nil
}
