package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// doctorCheck represents the result of checking a single environment variable.
type doctorCheck struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"` // masked for secrets
	OK    bool   `json:"ok"`
	Hint  string `json:"hint,omitempty"`
}

// runDoctorChecks verifies that all required environment variables are set.
// getenv is injected for testability.
func runDoctorChecks(getenv func(string) string) []doctorCheck {
	token := getenv("PAINTRESS_DISCORD_TOKEN")
	channelID := getenv("PAINTRESS_DISCORD_CHANNEL_ID")

	checks := []doctorCheck{
		checkEnvSecret("PAINTRESS_DISCORD_TOKEN", token),
		checkEnvRequired("PAINTRESS_DISCORD_CHANNEL_ID", channelID),
	}
	return checks
}

func checkEnvSecret(name, value string) doctorCheck {
	if value == "" {
		return doctorCheck{Name: name, Hint: "set this environment variable"}
	}
	return doctorCheck{Name: name, Value: maskSecret(value), OK: true}
}

func checkEnvRequired(name, value string) doctorCheck {
	if value == "" {
		return doctorCheck{Name: name, Hint: "set this environment variable"}
	}
	return doctorCheck{Name: name, Value: value, OK: true}
}

// maskSecret shows only the first 6 characters of a secret, replacing the rest with "***".
func maskSecret(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:6] + "***"
}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check required environment variables",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks := runDoctorChecks(os.Getenv)
			w := cmd.ErrOrStderr()

			fmt.Fprintln(w)
			fmt.Fprintln(w, "paintress-discord doctor")
			fmt.Fprintln(w)

			allOK := true
			for _, c := range checks {
				if c.OK {
					fmt.Fprintf(w, "  ✓  %-35s %s\n", c.Name, c.Value)
				} else {
					allOK = false
					hint := c.Hint
					if hint == "" {
						hint = "MISSING"
					}
					fmt.Fprintf(w, "  ✗  %-35s %s\n", c.Name, hint)
				}
			}
			fmt.Fprintln(w)

			if !allOK {
				return fmt.Errorf("some checks failed")
			}
			fmt.Fprintln(w, "All checks passed.")
			return nil
		},
	}
}
