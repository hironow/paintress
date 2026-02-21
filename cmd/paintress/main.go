package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/hironow/paintress/internal/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
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
		os.Exit(1)
	}
}
