package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/session"
	"github.com/spf13/cobra"
)

func newDeadLettersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dead-letters",
		Short: "Manage dead-lettered outbox items",
		Long:  "Inspect and purge outbox items that have exceeded the maximum retry count.",
	}

	cmd.AddCommand(newDeadLettersPurgeCommand())

	return cmd
}

func newDeadLettersPurgeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "purge [path]",
		Short: "Purge dead-lettered outbox items",
		Long: `Purge outbox items that have exceeded the maximum retry count (3+ failures).

By default runs in dry-run mode, showing the count of dead-lettered items.
Use --execute to perform actual deletion.`,
		Example: `  # Dry run: show dead-letter count (current directory)
  paintress dead-letters purge

  # Dry run: show count for a specific project
  paintress dead-letters purge /path/to/repo

  # Delete dead-lettered items
  paintress dead-letters purge --execute

  # Delete without confirmation prompt
  paintress dead-letters purge --execute --yes`,
		Args: cobra.MaximumNArgs(1),
		RunE: runDeadLettersPurge,
	}

	cmd.Flags().BoolP("execute", "x", false, "Execute purge (default: dry-run)")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDeadLettersPurge(cmd *cobra.Command, args []string) error {
	repoPath, err := resolveRepoPath(args)
	if err != nil {
		return err
	}

	store, err := session.NewOutboxStoreForDir(repoPath)
	if err != nil {
		return fmt.Errorf("open outbox store: %w", err)
	}
	defer store.Close()

	count, err := store.DeadLetterCount(cmd.Context())
	if err != nil {
		return err
	}

	execute, _ := cmd.Flags().GetBool("execute")
	outputFmt, _ := cmd.Flags().GetString("output")
	w := cmd.OutOrStdout()
	ew := cmd.ErrOrStderr()

	if outputFmt == "json" {
		out := struct {
			Count  int `json:"count"`
			Purged int `json:"purged"`
		}{Count: count}

		if execute && count > 0 {
			purged, purgeErr := store.PurgeDeadLetters(cmd.Context())
			if purgeErr != nil {
				return purgeErr
			}
			out.Purged = purged
		}
		data, jsonErr := json.Marshal(out)
		if jsonErr != nil {
			return jsonErr
		}
		fmt.Fprintln(w, string(data))
		return nil
	}

	// text output
	if count == 0 {
		fmt.Fprintln(ew, "No dead-lettered items.")
		return nil
	}

	fmt.Fprintf(ew, "%d dead-lettered outbox item(s).\n", count)

	if !execute {
		fmt.Fprintln(ew, "(dry-run — pass --execute to purge)")
		return nil
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Fprintf(ew, "\nPurge %d dead-lettered item(s)? [y/N] ", count)
		scanner := bufio.NewScanner(cmd.InOrStdin())
		if !scanner.Scan() {
			if scanErr := scanner.Err(); scanErr != nil {
				return fmt.Errorf("read confirmation: %w", scanErr)
			}
			fmt.Fprintln(ew, "Cancelled.")
			return nil
		}
		answer := strings.TrimSpace(scanner.Text())
		if answer != "y" && answer != "Y" {
			fmt.Fprintln(ew, "Cancelled.")
			return nil
		}
	}

	purged, purgeErr := store.PurgeDeadLetters(cmd.Context())
	if purgeErr != nil {
		return purgeErr
	}

	fmt.Fprintf(ew, "Purged %d dead-lettered item(s).\n", purged)

	return nil
}
