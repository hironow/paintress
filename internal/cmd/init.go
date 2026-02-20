package cmd

import (
	"fmt"
	"os"

	"github.com/hironow/paintress"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init <repo-path>",
		Short: "Initialize project configuration",
		Args:  cobra.ExactArgs(1),
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, args []string) error {
	repoPath := args[0]

	w := cmd.ErrOrStderr()
	fmt.Fprintln(w)
	fmt.Fprintf(w, "%s╔══════════════════════════════════════════════╗%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(w, "%s║          Paintress Init                      ║%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintf(w, "%s╚══════════════════════════════════════════════╝%s\n", paintress.ColorCyan, paintress.ColorReset)
	fmt.Fprintln(w)

	if err := paintress.RunInitWithReader(repoPath, os.Stdin); err != nil {
		return err
	}
	return nil
}
