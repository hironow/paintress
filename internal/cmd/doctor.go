package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor [repo-path]",
		Short: "Run health checks",
		Long: `Check environment health and tool availability.

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and docker.
When repo-path is provided (or defaults to current directory), also checks
.expedition/ structure, skills, config, and computes success rate metrics.`,
		Example: `  # Check current directory
  paintress doctor

  # Check a specific project directory
  paintress doctor /path/to/project

  # Machine-readable output
  paintress doctor -o json`,
		Args: cobra.MaximumNArgs(1),
		RunE: runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	outputFmt, _ := cmd.Flags().GetString("output")

	// Resolve continent: explicit arg or current working directory (aligned with amadeus/sightjack)
	continent, err := resolveRepoPath(args)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	claudeCmd := loadClaudeCmd(continent)
	checks := session.RunDoctor(claudeCmd, continent)

	hasFail := false
	for _, c := range checks {
		if c.Status == domain.CheckFail {
			hasFail = true
			break
		}
	}

	var metrics *domain.DoctorMetrics
	stateDir := filepath.Join(continent, domain.StateDir)
	eventStore := session.NewEventStore(stateDir, loggerFrom(cmd))
	metrics = usecase.ComputeSuccessRate(eventStore)

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
		if hasFail {
			return fmt.Errorf("some checks failed")
		}
		return nil
	}

	// text output — aligned with amadeus/sightjack format
	w := cmd.ErrOrStderr()
	fmt.Fprintln(w, "paintress doctor — environment health check")
	fmt.Fprintln(w)

	var fails, skips int
	for _, c := range checks {
		label := c.Status.StatusLabel()
		switch c.Status {
		case domain.CheckFail:
			fails++
		case domain.CheckSkip:
			skips++
		case domain.CheckWarn:
			skips++
		case domain.CheckOK:
			// no-op
		}

		msg := c.Message
		if msg == "" && c.Status == domain.CheckOK {
			msg = "OK"
		}

		fmt.Fprintf(w, "  [%-4s] %-16s %s\n", label, c.Name, msg)
		if c.Hint != "" {
			fmt.Fprintf(w, "         %-16s hint: %s\n", "", c.Hint)
		}
	}

	if metrics != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  success-rate: %s\n", metrics.SuccessRate)
	}

	fmt.Fprintln(w)
	if fails == 0 && skips == 0 {
		fmt.Fprintln(w, "All checks passed.")
		return nil
	}
	var msg string
	if fails > 0 {
		msg = fmt.Sprintf("%d check(s) failed", fails)
	}
	if skips > 0 {
		if msg != "" {
			msg += ", "
		}
		msg += fmt.Sprintf("%d skipped", skips)
	}
	fmt.Fprintln(w, msg+".")
	if fails > 0 {
		return fmt.Errorf("%d check(s) failed", fails)
	}
	return nil
}
