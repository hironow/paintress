package cmd

import (
	"strconv"
	"strings"
)

// knownBoolFlags lists every --flag that is registered as a BoolVar
// across all paintress subcommands. pflag treats bool flags specially
// (NoOptDefVal="true"), so `--flag false` does NOT consume "false" as
// the flag value. This rewriter normalises `--flag true/false/0/1/...`
// into `--flag=value` before cobra sees the args.
var knownBoolFlags = map[string]bool{
	"--dry-run": true,
	"--no-dev":  true,
	"--execute": true,
	"--verbose": true,
	"--json":    true,
	"--check":   true,
}

// RewriteBoolFlags normalises space-separated bool flag values into
// the `--flag=value` form that pflag expects. Non-bool flags and
// flags already using `=` are left untouched.
func RewriteBoolFlags(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Already has =, or not a flag at all → pass through.
		if !strings.HasPrefix(arg, "--") || strings.Contains(arg, "=") {
			out = append(out, arg)
			continue
		}

		if knownBoolFlags[arg] && i+1 < len(args) {
			next := args[i+1]
			if _, err := strconv.ParseBool(next); err == nil {
				out = append(out, arg+"="+next)
				i++ // consume the value
				continue
			}
		}

		out = append(out, arg)
	}
	return out
}
