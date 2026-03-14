package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

func newArchivePruneCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "archive-prune [path]",
		Short: "Prune old archived d-mails",
		Long: `Prune archived d-mail files older than a specified number of days.

By default runs in dry-run mode, listing candidates without deleting.
Use --execute to perform actual deletion. The archive/ directory is
git-tracked, so deletions should be reviewed and committed.`,
		Example: `  # Dry run: list files older than 30 days (current directory)
  paintress archive-prune

  # Dry run: list files for a specific project
  paintress archive-prune /path/to/repo

  # Delete files older than 14 days
  paintress archive-prune --days 14 --execute /path/to/repo

  # JSON output for scripting
  paintress archive-prune -o json /path/to/repo

  # Rebuild archive index from existing files
  paintress archive-prune --rebuild-index`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			days, _ := cmd.Flags().GetInt("days")
			if _, err := domain.NewDays(days); err != nil {
				return fmt.Errorf("--days: %w", err)
			}
			return nil
		},
		RunE: runArchivePrune,
	}

	cmd.Flags().IntP("days", "d", 30, "Retention days")
	cmd.Flags().BoolP("execute", "x", false, "Execute pruning (default: dry-run)")
	cmd.Flags().BoolP("dry-run", "n", false, "Dry-run mode (default behavior, explicit for scripting)")
	cmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().Bool("rebuild-index", false, "Rebuild archive index from existing files")

	return cmd
}

func runArchivePrune(cmd *cobra.Command, args []string) error {
	execute, _ := cmd.Flags().GetBool("execute")
	if execute && cmd.Flags().Changed("dry-run") {
		return fmt.Errorf("--execute and --dry-run are mutually exclusive")
	}
	repoPath, err := resolveRepoPath(args)
	if err != nil {
		return err
	}
	days, _ := cmd.Flags().GetInt("days")
	outputFmt, _ := cmd.Flags().GetString("output")
	stateDir := filepath.Join(repoPath, domain.StateDir)

	rebuildIndex, _ := cmd.Flags().GetBool("rebuild-index")
	if rebuildIndex {
		if execute {
			return fmt.Errorf("--rebuild-index cannot be combined with --execute")
		}
		if cmd.Flags().Changed("dry-run") {
			return fmt.Errorf("--rebuild-index cannot be combined with --dry-run")
		}
		indexPath := filepath.Join(stateDir, "archive", "index.jsonl")
		iw := &session.IndexWriter{}
		n, rbErr := iw.Rebuild(indexPath, stateDir, "paintress")
		if rbErr != nil {
			return fmt.Errorf("rebuild index: %w", rbErr)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Rebuilt index: %d entries → %s\n", n, indexPath)
		return nil
	}

	archiveOps := session.NewArchiveOps()

	rp, rpErr := domain.NewRepoPath(repoPath)
	if rpErr != nil {
		return rpErr
	}
	d, dErr := domain.NewDays(days)
	if dErr != nil {
		return dErr
	}
	// Collect archive candidates (dry-run to list only).
	archiveResult, err := usecase.ArchivePrune(domain.NewArchivePruneCommand(rp, d, false), archiveOps)
	if err != nil {
		return err
	}

	// Collect event file candidates.
	eventFiles, eventErr := archiveOps.ListExpiredEventFiles(cmd.Context(), stateDir, days)
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
			indexPaintressArchive(archiveResult.Candidates, stateDir, ew)

			execResult, execErr := archiveOps.ArchivePrune(repoPath, days, true)
			if execErr != nil {
				return execErr
			}
			out.Deleted = execResult.Deleted

			if len(eventFiles) > 0 {
				deleted, delErr := archiveOps.PruneEventFiles(cmd.Context(), stateDir, eventFiles)
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

	// text output — all metadata to stderr
	if totalCandidates == 0 {
		fmt.Fprintln(ew, "No files older than", days, "days.")
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

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Fprintf(ew, "\nDelete these %d file(s)? [y/N] ", totalCandidates)
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

	// Index archive candidates before deletion.
	indexPaintressArchive(archiveResult.Candidates, stateDir, ew)

	// Execute: archive deletion
	if len(archiveResult.Candidates) > 0 {
		execResult, execErr := archiveOps.ArchivePrune(repoPath, days, true)
		if execErr != nil {
			return execErr
		}
		fmt.Fprintf(ew, "Pruned %d archive file(s).\n", execResult.Deleted)
	}

	// Execute: event file deletion
	if len(eventFiles) > 0 {
		deleted, delErr := archiveOps.PruneEventFiles(cmd.Context(), stateDir, eventFiles)
		if delErr != nil {
			return fmt.Errorf("event prune failed: %w", delErr)
		}
		fmt.Fprintf(ew, "Pruned %d event file(s).\n", len(deleted))
	}

	// Prune flushed outbox DB rows + incremental vacuum.
	if pruned, pruneErr := archiveOps.PruneFlushedOutbox(cmd.Context(), repoPath); pruneErr == nil && pruned > 0 {
		fmt.Fprintf(ew, "Pruned %d flushed outbox row(s).\n", pruned)
	}

	if len(archiveResult.Candidates) > 0 {
		fmt.Fprintln(ew, "Note: archive/ is git-tracked. Run 'git status' to review and commit deletions.")
	}

	return nil
}

func indexPaintressArchive(candidates []string, stateDir string, w io.Writer) {
	archiveDir := filepath.Join(stateDir, "archive")
	var indexEntries []domain.IndexEntry
	for _, f := range candidates {
		if filepath.Ext(f) != ".md" {
			continue
		}
		fullPath := filepath.Join(archiveDir, f)
		indexEntries = append(indexEntries, session.ExtractMeta(fullPath, stateDir, "paintress"))
	}
	if len(indexEntries) == 0 {
		return
	}
	indexPath := filepath.Join(stateDir, "archive", "index.jsonl")
	iw := &session.IndexWriter{}
	if err := iw.Append(indexPath, indexEntries); err != nil {
		fmt.Fprintf(w, "Warning: index append: %v\n", err)
	} else {
		fmt.Fprintf(w, "Indexed %d entries → %s\n", len(indexEntries), indexPath)
	}
}
