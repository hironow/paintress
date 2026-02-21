package main

import (
	"fmt"
	"os"

	"github.com/hironow/paintress/internal/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	dir := "docs/cli"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir %s: %v\n", dir, err)
		os.Exit(1)
	}

	rootCmd := cmd.NewRootCommand()
	rootCmd.DisableAutoGenTag = true

	if err := doc.GenMarkdownTree(rootCmd, dir); err != nil {
		fmt.Fprintf(os.Stderr, "docgen: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Generated CLI docs in %s/\n", dir)
}
