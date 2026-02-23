package main

// placeholder.go provides minimal stubs for main.go until root.go is created.
// This file will be deleted when root.go is implemented (Task 4).

import (
	"fmt"

	"github.com/spf13/cobra"
)

// exitError wraps an exit code for the main function to handle.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}

// NewRootCommand is a stub that will be replaced by root.go.
func NewRootCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "paintress-slack",
		RunE: func(*cobra.Command, []string) error { return nil },
	}
}
