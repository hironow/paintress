package cmd

import (
	"fmt"

	"github.com/hironow/paintress"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check external command availability",
		Long: `Check that all external commands required by paintress are installed.

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and
docker. Reports version and path for each found command.`,
		Example: `  # Check all dependencies
  paintress doctor

  # Machine-readable output
  paintress doctor -o json`,
		Args: cobra.NoArgs,
		RunE: runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	outputFmt, _ := cmd.Flags().GetString("output")
	claudeCmd := paintress.DefaultClaudeCmd
	checks := paintress.RunDoctor(claudeCmd)

	allRequired := true
	for _, c := range checks {
		if c.Required && !c.OK {
			allRequired = false
			break
		}
	}

	if outputFmt == "json" {
		out, err := paintress.FormatDoctorJSON(checks)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), out)
		if !allRequired {
			return fmt.Errorf("some required commands are missing")
		}
		return nil
	}

	// text output
	w := cmd.ErrOrStderr()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s╔══════════════════════════════════════════════╗%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(w, "%s║          Paintress Doctor                    ║%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(w, "%s╚══════════════════════════════════════════════╝%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintln(w)

	allOK := true
	for _, c := range checks {
		if c.OK {
			fmt.Fprintf(w, "  %s✓%s  %-12s %s (%s)\n", paintress.ColorGreen, paintress.ColorReset, c.Name, c.Version, c.Path)
		} else {
			marker := "✗"
			color := paintress.ColorRed
			label := "MISSING (required)"
			if !c.Required {
				label = "not found (optional)"
				color = paintress.ColorYellow
			} else {
				allOK = false
			}
			fmt.Fprintf(w, "  %s%s%s  %-12s %s\n", color, marker, paintress.ColorReset, c.Name, label)
		}
	}
	fmt.Fprintln(w)

	if !allOK {
		return fmt.Errorf("some required commands are missing. Install them and try again")
	}
	fmt.Fprintln(w, "All checks passed.")
	return nil
}
