package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [repo-path]",
		Short: "Run the expedition loop",
		Long: `Run the expedition loop against a target repository.

Each expedition picks a Linear issue, creates a worktree branch,
invokes Claude Code to implement the change, opens a pull request,
and optionally runs a review cycle. The loop continues until
max-expeditions is reached or the issue queue is empty.

If repo-path is omitted, the current working directory is used.`,
		Example: `  # Run with defaults from current directory
  paintress run

  # Run with defaults (opus model, 50 expeditions, 33min timeout)
  paintress run /path/to/repo

  # Run with sonnet fallback and 3 parallel workers
  paintress run --model opus,sonnet --workers 3 /path/to/repo

  # Dry run (generate prompts only, no Claude invocation)
  paintress run --dry-run /path/to/repo

  # Skip dev server and use custom review command
  paintress run --no-dev --review-cmd "pnpm lint" /path/to/repo`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Set language global
			lang, _ := cmd.Flags().GetString("lang")
			if lang != "" {
				domain.Lang = lang
			}
			return nil
		},
		RunE: runExpedition,
	}

	cmd.Flags().Int("max-expeditions", 50, "Maximum number of expeditions")
	cmd.Flags().IntP("timeout", "t", 1980, "Timeout per expedition in seconds (default: 33min)")
	cmd.Flags().StringP("model", "m", "opus", "Model(s) comma-separated for reserve: opus,sonnet,haiku")
	cmd.Flags().StringP("base-branch", "b", "main", "Base branch")
	cmd.Flags().String("claude-cmd", domain.DefaultClaudeCmd, "Claude Code CLI command name")
	cmd.Flags().String("dev-cmd", "npm run dev", "Dev server command")
	cmd.Flags().String("dev-dir", "", "Dev server working directory (defaults to repo path)")
	cmd.Flags().String("dev-url", "http://localhost:3000", "Dev server URL")
	cmd.Flags().String("review-cmd", "", "Code review command after PR creation")
	cmd.Flags().IntP("workers", "w", 1, "Number of worktrees in pool (0 = direct execution)")
	cmd.Flags().String("setup-cmd", "", "Command to run after worktree creation (e.g. 'bun install')")
	cmd.Flags().Bool("no-dev", false, "Skip dev server startup")
	cmd.Flags().BoolP("dry-run", "n", false, "Generate prompts only")
	cmd.Flags().String("notify-cmd", "", "Notification command ({title}, {message} placeholders)")
	cmd.Flags().String("approve-cmd", "", "Approval command ({message} placeholder, exit 0 = approve)")
	cmd.Flags().Bool("auto-approve", false, "Skip approval gate for HIGH severity D-Mail")

	return cmd
}

// configFromProject builds a runtime Config from a ProjectConfig.
// Runtime-only fields (Continent, DryRun, OutputFormat) are set by the caller.
func configFromProject(pc *domain.ProjectConfig) domain.Config {
	return domain.Config{
		MaxExpeditions: pc.MaxExpeditions,
		TimeoutSec:     pc.TimeoutSec,
		Model:          pc.Model,
		BaseBranch:     pc.BaseBranch,
		ClaudeCmd:      pc.ClaudeCmd,
		DevCmd:         pc.DevCmd,
		DevDir:         pc.DevDir,
		DevURL:         pc.DevURL,
		ReviewCmd:      pc.ReviewCmd,
		Workers:        pc.Workers,
		SetupCmd:       pc.SetupCmd,
		NoDev:          pc.NoDev,
		NotifyCmd:      pc.NotifyCmd,
		ApproveCmd:     pc.ApproveCmd,
		AutoApprove:    pc.AutoApprove,
		MaxRetries:     pc.MaxRetries,
	}
}

func runExpedition(cmd *cobra.Command, args []string) error {
	continent, err := resolveRepoPath(args)
	if err != nil {
		return err
	}

	// Pre-flight check: ensure init has been run
	cfgPath := domain.ProjectConfigPath(continent)
	if _, statErr := os.Stat(cfgPath); statErr != nil {
		return fmt.Errorf("not initialized — run 'paintress init %s' first", continent)
	}

	// Preflight: verify required binaries exist
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	claudeCmd, _ := cmd.Flags().GetString("claude-cmd")
	bins := []string{"git"}
	if !dryRun {
		bins = append(bins, claudeCmd)
	}
	if err := session.PreflightCheck(bins...); err != nil {
		return err
	}

	// Preflight: verify git remote exists (required for PR creation)
	if !dryRun {
		if err := session.PreflightCheckRemote(continent); err != nil {
			return err
		}
	}

	// Load project config as base, then override with explicit CLI flags
	projectCfg, pcErr := session.LoadProjectConfig(continent)
	if pcErr != nil {
		return pcErr
	}
	cfg := configFromProject(projectCfg)
	cfg.Continent = continent
	cfg.DryRun, _ = cmd.Flags().GetBool("dry-run")
	cfg.OutputFormat, _ = cmd.Flags().GetString("output")

	// Override with explicitly-set CLI flags only
	if cmd.Flags().Changed("max-expeditions") {
		cfg.MaxExpeditions, _ = cmd.Flags().GetInt("max-expeditions")
	}
	if cmd.Flags().Changed("timeout") {
		cfg.TimeoutSec, _ = cmd.Flags().GetInt("timeout")
	}
	if cmd.Flags().Changed("model") {
		cfg.Model, _ = cmd.Flags().GetString("model")
	}
	if cmd.Flags().Changed("base-branch") {
		cfg.BaseBranch, _ = cmd.Flags().GetString("base-branch")
	}
	if cmd.Flags().Changed("claude-cmd") {
		cfg.ClaudeCmd, _ = cmd.Flags().GetString("claude-cmd")
	}
	if cmd.Flags().Changed("dev-cmd") {
		cfg.DevCmd, _ = cmd.Flags().GetString("dev-cmd")
	}
	if cmd.Flags().Changed("dev-dir") {
		cfg.DevDir, _ = cmd.Flags().GetString("dev-dir")
	}
	if cmd.Flags().Changed("dev-url") {
		cfg.DevURL, _ = cmd.Flags().GetString("dev-url")
	}
	if cmd.Flags().Changed("review-cmd") {
		cfg.ReviewCmd, _ = cmd.Flags().GetString("review-cmd")
	}
	if cmd.Flags().Changed("workers") {
		cfg.Workers, _ = cmd.Flags().GetInt("workers")
	}
	if cmd.Flags().Changed("setup-cmd") {
		cfg.SetupCmd, _ = cmd.Flags().GetString("setup-cmd")
	}
	if cmd.Flags().Changed("no-dev") {
		cfg.NoDev, _ = cmd.Flags().GetBool("no-dev")
	}
	if cmd.Flags().Changed("notify-cmd") {
		cfg.NotifyCmd, _ = cmd.Flags().GetString("notify-cmd")
	}
	if cmd.Flags().Changed("approve-cmd") {
		cfg.ApproveCmd, _ = cmd.Flags().GetString("approve-cmd")
	}
	if cmd.Flags().Changed("auto-approve") {
		cfg.AutoApprove, _ = cmd.Flags().GetBool("auto-approve")
	}

	// Derive review-cmd from base-branch if neither CLI nor config set it
	if cfg.ReviewCmd == "" {
		cfg.ReviewCmd = fmt.Sprintf("codex review --base %s", cfg.BaseBranch)
	}

	// Set language global: CLI flag > config > fallback "ja"
	lang, _ := cmd.Flags().GetString("lang")
	if lang == "" {
		lang = projectCfg.Lang
	}
	if lang == "" {
		lang = "ja"
	}
	domain.Lang = lang

	logger := loggerFrom(cmd)
	stateDir := filepath.Join(continent, domain.StateDir)
	eventStore := session.NewEventStore(stateDir, logger)

	if err := session.ValidateContinent(cfg.Continent, logger); err != nil {
		return err
	}

	// Use command's context (set by ExecuteContext in main)
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, stop := signal.NotifyContext(ctx, shutdownSignals...)
	defer stop()

	notifier := session.BuildNotifier(cfg.NotifyCmd)
	p := session.NewPaintress(cfg, logger, cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd.InOrStdin(), nil)
	rp, rpErr := domain.NewRepoPath(continent)
	if rpErr != nil {
		return rpErr
	}
	exitCode, ucErr := usecase.RunExpeditions(ctx, domain.NewRunExpeditionCommand(rp), p, eventStore, logger, notifier, &platform.OTelPolicyMetrics{})
	if ucErr != nil {
		return ucErr
	}
	if exitCode != 0 {
		return &ExitError{Code: exitCode, Err: fmt.Errorf("expedition exited with code %d", exitCode)}
	}
	return nil
}
