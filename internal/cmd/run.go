package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hironow/paintress"
	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <repo-path>",
		Short: "Run the expedition loop",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Derive --review-cmd from --base-branch when not explicitly set
			if !cmd.Flags().Changed("review-cmd") {
				baseBranch, _ := cmd.Flags().GetString("base-branch")
				cmd.Flags().Set("review-cmd", fmt.Sprintf("codex review --base %s", baseBranch))
			}

			// Set language global
			lang, _ := cmd.Flags().GetString("lang")
			if lang == "ja" || lang == "en" || lang == "fr" {
				paintress.Lang = lang
			}

			return nil
		},
		RunE: runExpedition,
	}

	cmd.Flags().Int("max-expeditions", 50, "Maximum number of expeditions")
	cmd.Flags().Int("timeout", 1980, "Timeout per expedition in seconds (default: 33min)")
	cmd.Flags().String("model", "opus", "Model(s) comma-separated for reserve: opus,sonnet,haiku")
	cmd.Flags().String("base-branch", "main", "Base branch")
	cmd.Flags().String("claude-cmd", paintress.DefaultClaudeCmd, "Claude Code CLI command name")
	cmd.Flags().String("dev-cmd", "npm run dev", "Dev server command")
	cmd.Flags().String("dev-dir", "", "Dev server working directory (defaults to repo path)")
	cmd.Flags().String("dev-url", "http://localhost:3000", "Dev server URL")
	cmd.Flags().String("review-cmd", "", "Code review command after PR creation")
	cmd.Flags().Int("workers", 1, "Number of worktrees in pool (0 = direct execution)")
	cmd.Flags().String("setup-cmd", "", "Command to run after worktree creation (e.g. 'bun install')")
	cmd.Flags().Bool("no-dev", false, "Skip dev server startup")
	cmd.Flags().Bool("dry-run", false, "Generate prompts only")

	return cmd
}

func runExpedition(cmd *cobra.Command, args []string) error {
	repoPath := args[0]
	continent, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	cfg := paintress.Config{}
	cfg.Continent = continent
	cfg.MaxExpeditions, _ = cmd.Flags().GetInt("max-expeditions")
	cfg.TimeoutSec, _ = cmd.Flags().GetInt("timeout")
	cfg.Model, _ = cmd.Flags().GetString("model")
	cfg.BaseBranch, _ = cmd.Flags().GetString("base-branch")
	cfg.ClaudeCmd, _ = cmd.Flags().GetString("claude-cmd")
	cfg.DevCmd, _ = cmd.Flags().GetString("dev-cmd")
	cfg.DevDir, _ = cmd.Flags().GetString("dev-dir")
	cfg.DevURL, _ = cmd.Flags().GetString("dev-url")
	cfg.ReviewCmd, _ = cmd.Flags().GetString("review-cmd")
	cfg.Workers, _ = cmd.Flags().GetInt("workers")
	cfg.SetupCmd, _ = cmd.Flags().GetString("setup-cmd")
	cfg.NoDev, _ = cmd.Flags().GetBool("no-dev")
	cfg.DryRun, _ = cmd.Flags().GetBool("dry-run")
	cfg.OutputFormat, _ = cmd.Flags().GetString("output")

	shutdownTracer := paintress.InitTracer("paintress", Version)
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		shutdownTracer(shutdownCtx)
	}()

	if err := paintress.ValidateContinent(cfg.Continent); err != nil {
		return err
	}

	// Use command's context (set by ExecuteContext in main)
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		paintress.LogWarn("%s", fmt.Sprintf(paintress.Msg("signal_received"), sig))
		cancel()
	}()

	p := paintress.NewPaintress(cfg)
	exitCode := p.Run(ctx)
	if exitCode != 0 {
		return fmt.Errorf("expedition exited with code %d", exitCode)
	}
	return nil
}
