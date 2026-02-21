package cmd

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

func newUpdateCommand() *cobra.Command {
	var checkOnly bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Self-update paintress to the latest release",
		Long: `Self-update paintress to the latest GitHub release.

Downloads the latest release, verifies the checksum, and replaces
the current binary. Use --check to only check for updates without
installing.`,
		Example: `  # Check for updates
  paintress update --check

  # Update to the latest version
  paintress update`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			updater, err := selfupdate.NewUpdater(selfupdate.Config{
				Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
			})
			if err != nil {
				return fmt.Errorf("failed to create updater: %w", err)
			}

			latest, found, err := updater.DetectLatest(cmd.Context(), selfupdate.ParseSlug("hironow/paintress"))
			if err != nil {
				return fmt.Errorf("failed to detect latest version: %w", err)
			}
			if !found {
				fmt.Fprintln(cmd.OutOrStdout(), "No release found.")
				return nil
			}

			// Guard: version may be "dev" for local builds (non-semver).
			ver := strings.TrimPrefix(Version, "v")
			if _, err := semver.NewVersion(ver); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Development build (version %q) — cannot compare versions.\nLatest release: v%s\n", Version, latest.Version())
				return nil
			}

			if latest.LessOrEqual(Version) {
				fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (v%s).\n", ver)
				return nil
			}

			if checkOnly {
				fmt.Fprintf(cmd.OutOrStdout(), "Update available: v%s → v%s\n", ver, latest.Version())
				return nil
			}

			exe, err := selfupdate.ExecutablePath()
			if err != nil {
				return fmt.Errorf("failed to locate executable: %w", err)
			}

			if err := updater.UpdateTo(cmd.Context(), latest, exe); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated to v%s\n", latest.Version())
			return nil
		},
	}

	cmd.Flags().BoolVar(&checkOnly, "check", false, "Check for updates without installing")

	return cmd
}
