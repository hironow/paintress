package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"

	cmd "github.com/hironow/paintress/internal/cmd"
	"github.com/hironow/paintress/internal/domain"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals...)
	defer stop()

	rootCmd := cmd.NewRootCommand()
	args := os.Args[1:]
	if cmd.NeedsDefaultRun(rootCmd, args) {
		args = append([]string{"run"}, args...)
	}
	rootCmd.SetArgs(args)

	err := rootCmd.ExecuteContext(ctx)
	return handleError(err, os.Stderr)
}

// handleError processes an error from command execution, printing to w only
// when the error is not silent. Returns the appropriate exit code.
func handleError(err error, w io.Writer) int {
	if err != nil {
		var exitErr *cmd.ExitError
		var silent *domain.SilentError
		if !errors.As(err, &exitErr) && !errors.As(err, &silent) {
			fmt.Fprintf(w, "error: %v\n", err)
		}
	}
	return domain.ExitCode(err)
}
