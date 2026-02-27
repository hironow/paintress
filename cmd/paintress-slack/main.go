package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(),
		shutdownSignals...)
	defer stop()

	rootCmd := NewRootCommand()
	rootCmd.SetArgs(os.Args[1:])

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		var exitErr *exitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.code)
		}
		os.Exit(1)
	}
}
