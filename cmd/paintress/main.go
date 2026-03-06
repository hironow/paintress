package main

import (
	"context"
	"errors"
	"os"
	"os/signal"

	"github.com/hironow/paintress/internal/cmd"
	"github.com/hironow/paintress/internal/domain"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		shutdownSignals...)
	defer stop()

	rootCmd := cmd.NewRootCommand()

	// NOTE: RewriteBoolFlags was intentionally removed (MY-334).
	// Space-separated bool values (e.g. `--dry-run false`) are no longer
	// normalised. Use `--dry-run=false` instead. This aligns with
	// POSIX/GNU conventions and matches kubectl/gh/docker behaviour.

	// Preserve old `paintress [flags] <repo>` shorthand:
	// prepend "run" when no subcommand is specified.
	args := os.Args[1:]
	if cmd.NeedsDefaultRun(rootCmd, args) {
		args = append([]string{"run"}, args...)
	}
	rootCmd.SetArgs(args)

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(domain.ExitCode(err))
	}
}
