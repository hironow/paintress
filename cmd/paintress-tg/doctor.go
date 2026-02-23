package main

import (
	"fmt"
	"os"
	"strconv"

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
	token := getenv("PAINTRESS_TG_TOKEN")
	chatID := getenv("PAINTRESS_TG_CHAT_ID")

	checks := []doctorCheck{
		checkEnvSecret("PAINTRESS_TG_TOKEN", token),
		checkChatID(chatID),
	}
	return checks
}

func checkEnvSecret(name, value string) doctorCheck {
	if value == "" {
		return doctorCheck{Name: name, Hint: "set this environment variable"}
	}
	return doctorCheck{Name: name, Value: maskSecret(value), OK: true}
}

func checkChatID(value string) doctorCheck {
	name := "PAINTRESS_TG_CHAT_ID"
	if value == "" {
		return doctorCheck{Name: name, Hint: "set this environment variable"}
	}
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		return doctorCheck{Name: name, Value: value, Hint: "must be an integer"}
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

const setupGuide = `
Setup Guide:

  PAINTRESS_TG_TOKEN — Telegram Bot API token
    1. Open Telegram and search for @BotFather
    2. Send /newbot and follow the prompts (name + username ending in "bot")
    3. BotFather replies with your token (format: 123456:ABC-DEF...)
    4. To retrieve later: send /mybots to @BotFather, select your bot, tap "API Token"

  PAINTRESS_TG_CHAT_ID — Target chat ID (integer)
    1. Add your bot to the target chat (group or channel)
    2. Send any message in that chat
    3. Open: https://api.telegram.org/bot<TOKEN>/getUpdates
    4. Find "chat":{"id": ...} in the JSON response
    Alternative: send /start to @RawDataBot to get your personal chat ID

  Example:
    export PAINTRESS_TG_TOKEN="123456:ABC-DEF..."
    export PAINTRESS_TG_CHAT_ID="987654321"
`

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "doctor",
		Short:        "Check required environment variables",
		SilenceUsage: true,
		Args:         cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks := runDoctorChecks(os.Getenv)
			w := cmd.ErrOrStderr()

			fmt.Fprintln(w)
			fmt.Fprintln(w, "paintress-tg doctor")
			fmt.Fprintln(w)

			allOK := true
			for _, c := range checks {
				if c.OK {
					fmt.Fprintf(w, "  ✓  %-30s %s\n", c.Name, c.Value)
				} else {
					allOK = false
					hint := c.Hint
					if hint == "" {
						hint = "MISSING"
					}
					fmt.Fprintf(w, "  ✗  %-30s %s\n", c.Name, hint)
				}
			}
			fmt.Fprintln(w)

			if !allOK {
				fmt.Fprint(w, setupGuide)
				return fmt.Errorf("some checks failed")
			}
			fmt.Fprintln(w, "All checks passed.")
			return nil
		},
	}
}
