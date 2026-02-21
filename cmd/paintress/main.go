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

	// Normalise bool flags: `--dry-run false` → `--dry-run=false`
	// (pflag's NoOptDefVal prevents space-separated bool values).
	args := cmd.RewriteBoolFlags(os.Args[1:])

	// Preserve old `paintress [flags] <repo>` shorthand:
	// prepend "run" when no subcommand is specified.
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
