package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

// doctorJSONCheck is the JSON-serializable form of DoctorCheck for cmd output.
type doctorJSONCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "OK", "FAIL", "WARN", "SKIP", "FIX"
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

// doctorJSONOutput is the JSON-serializable doctor output.
type doctorJSONOutput struct {
	Checks  []doctorJSONCheck     `json:"checks"`
	Metrics *domain.DoctorMetrics `json:"metrics,omitempty"`
}

func newDoctorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor [path]",
		Short: "Run health checks",
		Long: `Check environment health and tool availability.

Verifies: git, claude (Claude Code CLI), gh (GitHub CLI), and docker.
When path is provided (or defaults to current directory), also checks
.expedition/ structure, skills, config, and computes success rate metrics.
Each check reports one of four statuses: OK (passed), FAIL (exit 1),
SKIP (dependency missing), WARN (advisory, exit 0).

The context-budget check estimates token consumption per category
(tools, skills, plugins, mcp, hooks) and marks the heaviest.
When the threshold (20,000 tokens) is exceeded, a category-specific
hint recommends adjusting .claude/settings.json.

Use --repair to auto-fix repairable issues (stale PID, missing SKILL.md, etc.).`,
		Example: `  # Check current directory
  paintress doctor

  # Check a specific project directory
  paintress doctor /path/to/project

  # Machine-readable output
  paintress doctor -o json

  # Auto-fix repairable issues
  paintress doctor --repair`,
		Args: cobra.MaximumNArgs(1),
		RunE: runDoctor,
	}
	cmd.Flags().Bool("repair", false, "Auto-fix repairable issues")
	return cmd
}

func runDoctor(cmd *cobra.Command, args []string) error {
	outputFmt, _ := cmd.Flags().GetString("output")

	// Resolve continent: explicit arg or current working directory (aligned with amadeus/sightjack)
	continent, err := resolveRepoPath(args)
	if err != nil {
		return fmt.Errorf("resolve repo path: %w", err)
	}

	claudeCmd := loadClaudeCmd(continent)
	repair, _ := cmd.Flags().GetBool("repair")
	linearFlag, _ := cmd.Flags().GetBool("linear")
	mode := domain.NewTrackingMode(linearFlag)
	checks := session.RunDoctor(claudeCmd, continent, repair, mode)

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
	metrics = usecase.ComputeSuccessRate(cmd.Context(), eventStore, loggerFrom(cmd))

	if outputFmt == "json" {
		jsonChecks := make([]doctorJSONCheck, len(checks))
		for i, c := range checks {
			jsonChecks[i] = doctorJSONCheck{
				Name:    c.Name,
				Status:  c.Status.StatusLabel(),
				Message: c.Message,
				Hint:    c.Hint,
			}
		}
		output := doctorJSONOutput{Checks: jsonChecks, Metrics: metrics}
		data, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return fmt.Errorf("format doctor JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		if hasFail {
			return &domain.SilentError{Err: fmt.Errorf("some checks failed")}
		}
		return nil
	}

	// text output — aligned with amadeus/sightjack format
	w := cmd.ErrOrStderr()
	logger := platform.NewLogger(w, false)
	fmt.Fprintln(w, "paintress doctor — environment health check")
	fmt.Fprintln(w)

	var fails, skips, warns int
	for _, c := range checks {
		label := logger.Colorize(fmt.Sprintf("%-4s", c.Status.StatusLabel()), platform.StatusColor(c.Status))
		fmt.Fprintf(w, "  [%s] %-16s %s\n", label, c.Name, c.Message)
		if c.Hint != "" {
			fmt.Fprintf(w, "         %-16s hint: %s\n", "", c.Hint)
		}
		switch c.Status {
		case domain.CheckFail:
			fails++
		case domain.CheckSkip:
			skips++
		case domain.CheckWarn:
			warns++
		}
	}

	if metrics != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  success-rate: %s\n", metrics.SuccessRate)
	}

	fmt.Fprintln(w)
	if fails == 0 && skips == 0 && warns == 0 {
		fmt.Fprintln(w, "All checks passed.")
		return nil
	}
	var parts []string
	if fails > 0 {
		parts = append(parts, fmt.Sprintf("%d check(s) failed", fails))
	}
	if warns > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s)", warns))
	}
	if skips > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", skips))
	}
	fmt.Fprintln(w, strings.Join(parts, ", ")+".")
	if fails > 0 {
		return &domain.SilentError{Err: fmt.Errorf("%d check(s) failed", fails)}
	}
	return nil
}
