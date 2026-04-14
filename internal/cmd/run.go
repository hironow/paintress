package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase"
	"github.com/hironow/paintress/internal/usecase/port"
	"github.com/spf13/cobra"
)

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [path]",
		Short: "Run the expedition loop",
		Long: `Run the expedition loop against a target repository.

Each expedition picks a Linear issue, creates a worktree branch,
invokes Claude Code to implement the change, opens a pull request,
and optionally runs a review cycle. The loop continues until
max-expeditions is reached or the issue queue is empty.

If path is omitted, the current working directory is used.`,
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
	cmd.Flags().Duration("idle-timeout", domain.DefaultIdleTimeout, "D-Mail waiting phase timeout (0 = 24h safety cap, negative = disable waiting)")

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
		IdleTimeout:    pc.IdleTimeout,
	}
}

func runExpedition(cmd *cobra.Command, args []string) error {
	continent, err := resolveTargetDir(args)
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
	if cmd.Flags().Changed("idle-timeout") {
		cfg.IdleTimeout, _ = cmd.Flags().GetDuration("idle-timeout")
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

	// Initialize process-wide circuit breaker for rate limit / server error protection
	session.SetCircuitBreaker(platform.NewCircuitBreaker(logger))

	stateDir := filepath.Join(continent, domain.StateDir)
	eventStore := session.NewEventStore(stateDir, logger)

	// Cutover wiring: ensure SeqNr allocation is active
	seqCounter, cutoverErr := session.EnsureCutover(cmd.Context(), stateDir, "paintress.state", logger)
	if cutoverErr != nil {
		return fmt.Errorf("cutover: %w", cutoverErr)
	}
	defer seqCounter.Close()

	if _, err := session.ValidateContinent(cfg.Continent, logger); err != nil {
		return err
	}

	// Use command's context (set by ExecuteContext in main).
	// Signal handling is done in main.go's two-context pattern;
	// do NOT register signals here to avoid double-handling.
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	notifier := session.BuildNotifier(cfg.NotifyCmd)
	approver := session.BuildApprover(cfg, cmd.InOrStdin(), cmd.ErrOrStderr())
	p := session.NewPaintress(cfg, logger, cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd.InOrStdin(), nil, approver, domain.NewExpeditionAggregate())
	defer p.CloseRunner()
	p.SetSeqAllocator(seqCounter)
	p.SetCheckpointScanner(session.NewCheckpointScanner(continent))
	if _, rpErr := domain.NewRepoPath(continent); rpErr != nil {
		return rpErr
	}

	// Resolve tracking mode from --linear flag
	linearFlag, _ := cmd.Flags().GetBool("linear")
	mode := domain.NewTrackingMode(linearFlag)

	// Build target provider for wave mode
	var targetProvider port.TargetProvider
	if mode.IsWave() {
		stepProgressReader := session.NewStepProgressReader(eventStore, logger)
		targetProvider = usecase.NewWaveTargetProvider(stepProgressReader, session.NewInboxReader(continent))
	}

	logger.Info("paintress run: starting initial expedition cycle (mode=%s)...", mode)
	usecase.PrepareExpeditionRunner(ctx, p, eventStore, logger, notifier, &platform.OTelPolicyMetrics{}, session.NewInboxArchiver(nil), p, cfg.Continent, cfg.MaxRetries, mode, targetProvider)
	exitCode := p.Run(ctx)
	var ucErr error
	if ucErr != nil {
		summary := p.HandoverSummary()
		return tryWriteHandover(ctx, ucErr, continent, domain.HandoverState{
			Tool:       "paintress",
			Operation:  "expedition",
			InProgress: "expedition cycle (initial)",
			Completed: []string{
				fmt.Sprintf("%d expeditions attempted, %d succeeded", summary.Total, summary.Success),
			},
			Remaining: []string{
				fmt.Sprintf("%d of %d max expeditions remaining", int64(cfg.MaxExpeditions)-summary.Total, cfg.MaxExpeditions),
			},
			PartialState: map[string]string{
				"BaseBranch": cfg.BaseBranch,
				"Model":      cfg.Model,
				"Workers":    fmt.Sprintf("%d", cfg.Workers),
				"MaxExp":     fmt.Sprintf("%d", cfg.MaxExpeditions),
			},
		}, logger)
	}
	if exitCode != 0 {
		return &ExitError{Code: exitCode, Err: fmt.Errorf("expedition exited with code %d", exitCode)}
	}
	logger.Info("paintress run: initial expedition cycle completed (exit code %d)", exitCode)

	// Skip waiting in dry-run mode or when explicitly disabled
	if cfg.DryRun || cfg.IdleTimeout < 0 {
		return nil
	}

	// Start inbox monitor for waiting phase
	inboxCh, monErr := session.MonitorInbox(ctx, continent, logger)
	if monErr != nil {
		return fmt.Errorf("inbox monitor: %w", monErr)
	}

	// Waiting loop: wait for D-Mail → re-run expeditions → repeat
	for {
		arrivedDMail, waitErr := session.WaitForDMail(ctx, inboxCh, cfg.IdleTimeout, logger)
		if waitErr != nil {
			return tryWriteHandover(ctx, waitErr, continent, domain.HandoverState{
				Tool:         "paintress",
				Operation:    "expedition",
				InProgress:   "D-Mail waiting phase",
				Completed:    []string{"Initial expedition cycle completed"},
				PartialState: map[string]string{"Phase": "waiting"},
			}, logger)
		}
		if arrivedDMail == nil {
			writeHandoverOnCancel(ctx, continent, domain.HandoverState{
				Tool:         "paintress",
				Operation:    "expedition",
				InProgress:   "D-Mail waiting phase (clean exit on Ctrl+C)",
				Completed:    []string{"Initial expedition cycle completed"},
				PartialState: map[string]string{"Phase": "waiting-cancelled"},
			}, logger)
			return nil
		}

		// Re-run expeditions on D-Mail arrival
		logger.Info("paintress run: D-Mail received (kind=%s, name=%s), re-running expedition cycle...", arrivedDMail.Kind, arrivedDMail.Name)
		p.CloseRunner() // close previous iteration's session store
		p = session.NewPaintress(cfg, logger, cmd.OutOrStdout(), cmd.ErrOrStderr(), cmd.InOrStdin(), nil, approver, domain.NewExpeditionAggregate())
		defer p.CloseRunner()
		p.SetSeqAllocator(seqCounter)
		p.SetCheckpointScanner(session.NewCheckpointScanner(continent))
		usecase.PrepareExpeditionRunner(ctx, p, eventStore, logger, notifier, &platform.OTelPolicyMetrics{}, session.NewInboxArchiver(nil), p, cfg.Continent, cfg.MaxRetries, mode, targetProvider)
		exitCode = p.Run(ctx)
		ucErr = nil
		if ucErr != nil {
			summary := p.HandoverSummary()
			return tryWriteHandover(ctx, ucErr, continent, domain.HandoverState{
				Tool:       "paintress",
				Operation:  "expedition",
				InProgress: "expedition cycle (D-Mail re-run)",
				Completed: []string{
					"D-Mail received and re-run started",
					fmt.Sprintf("%d expeditions attempted, %d succeeded", summary.Total, summary.Success),
				},
				Remaining: []string{
					fmt.Sprintf("%d of %d max expeditions remaining", int64(cfg.MaxExpeditions)-summary.Total, cfg.MaxExpeditions),
				},
				PartialState: map[string]string{
					"BaseBranch": cfg.BaseBranch,
					"Model":      cfg.Model,
					"Workers":    fmt.Sprintf("%d", cfg.Workers),
				},
			}, logger)
		}
		if exitCode != 0 {
			return &ExitError{Code: exitCode, Err: fmt.Errorf("expedition exited with code %d", exitCode)}
		}
	}
}
