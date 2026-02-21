package cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// NeedsDefaultRun reports whether args should be prefixed with "run"
// to preserve the old `paintress [flags] <repo>` shorthand. It scans
// past known root persistent flags to find the first positional arg
// and checks whether it is a registered subcommand.
func NeedsDefaultRun(rootCmd *cobra.Command, args []string) bool {
	if len(args) == 0 {
		return false
	}

	// --version and --help are "exit early" flags handled by cobra's root.
	// Never rewrite args when these are present, regardless of other args.
	for _, a := range args {
		if a == "--version" || a == "--help" || a == "-h" {
			return false
		}
		if a == "--" {
			break
		}
	}

	first := args[0]

	// Not a flag → check if first arg is a known subcommand
	if !strings.HasPrefix(first, "-") {
		return !isSubcommand(rootCmd, first)
	}

	// Build lookup tables from root's persistent flags.
	// --version and --help are auto-added by cobra after Execute starts,
	// so we hard-code them here.
	boolFlags := map[string]bool{
		"--help": true, "-h": true,
		"--version": true,
	}
	stringFlags := map[string]bool{}

	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Value.Type() == "bool" {
			boolFlags["--"+f.Name] = true
			if f.Shorthand != "" {
				boolFlags["-"+f.Shorthand] = true
			}
		} else {
			stringFlags["--"+f.Name] = true
			if f.Shorthand != "" {
				stringFlags["-"+f.Shorthand] = true
			}
		}
	})

	// Scan args: skip known root flags, find first positional arg
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if strings.HasPrefix(arg, "-") {
			// --flag=value is self-contained
			if strings.Contains(arg, "=") {
				continue
			}
			if boolFlags[arg] {
				continue
			}
			if stringFlags[arg] {
				i++ // skip the value
				continue
			}
			// Unknown flag → must be run-specific
			return true
		}

		// First positional arg
		return !isSubcommand(rootCmd, arg)
	}

	return false // only root flags, no positional
}

func isSubcommand(rootCmd *cobra.Command, name string) bool {
	for _, c := range rootCmd.Commands() {
		if c.Name() == name {
			return true
		}
		for _, a := range c.Aliases {
			if a == name {
				return true
			}
		}
	}
	return false
}
