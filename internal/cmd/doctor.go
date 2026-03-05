package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor [repo-path]",
		Short: "Check external command availability",
		Long: `Check that all external commands required by paintress are installed.

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and
docker. Reports version and path for each found command.

If repo-path is provided, also computes expedition success rate metrics.`,
		Example: `  # Check all dependencies
  paintress doctor

  # Machine-readable output
  paintress doctor -o json

  # Include repo metrics
  paintress doctor -o json ./my-repo`,
		Args: cobra.MaximumNArgs(1),
		RunE: runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	outputFmt, _ := cmd.Flags().GetString("output")
	claudeCmd := platform.DefaultClaudeCmd
	var continent string
	if len(args) > 0 {
		continent = args[0]
	}
	checks := session.RunDoctor(claudeCmd, continent)

	allRequired := true
	for _, c := range checks {
		if c.Required && !c.OK {
			allRequired = false
			break
		}
	}

	var metrics *domain.DoctorMetrics
	if len(args) > 0 {
		stateDir := filepath.Join(args[0], ".expedition")
		eventStore := session.NewEventStore(stateDir, loggerFrom(cmd))
		metrics = usecase.ComputeSuccessRate(eventStore)
	}

	if outputFmt == "json" {
		output := domain.DoctorOutput{
			Checks:  checks,
			Metrics: metrics,
		}
		out, err := domain.FormatDoctorOutputJSON(output)
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
	fmt.Fprintln(w, "╔══════════════════════════════════════════════╗")
	fmt.Fprintln(w, "║          Paintress Doctor                    ║")
	fmt.Fprintln(w, "╚══════════════════════════════════════════════╝")
	fmt.Fprintln(w)

	allOK := true
	for _, c := range checks {
		if c.OK {
			fmt.Fprintf(w, "  ✓  %-12s %s (%s)\n", c.Name, c.Version, c.Path)
		} else {
			marker := "✗"
			label := "MISSING (required)"
			if !c.Required {
				label = "not found (optional)"
				if c.Version != "" {
					label = c.Version + " (optional)"
				}
			} else {
				allOK = false
			}
			fmt.Fprintf(w, "  %s  %-12s %s\n", marker, c.Name, label)
			if c.Hint != "" {
				fmt.Fprintf(w, "         %-12s hint: %s\n", "", c.Hint)
			}
		}
	}

	if metrics != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  success-rate: %s\n", metrics.SuccessRate)
	}

	fmt.Fprintln(w)
	if !allOK {
		return fmt.Errorf("some required commands are missing. Install them and try again")
	}
	fmt.Fprintln(w, "All checks passed.")
	return nil
}
