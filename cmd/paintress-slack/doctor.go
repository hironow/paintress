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
	token := getenv("PAINTRESS_SLACK_TOKEN")
	channelID := getenv("PAINTRESS_SLACK_CHANNEL_ID")
	appToken := getenv("PAINTRESS_SLACK_APP_TOKEN")

	checks := []doctorCheck{
		checkEnvSecret("PAINTRESS_SLACK_TOKEN", token),
		checkEnvRequired("PAINTRESS_SLACK_CHANNEL_ID", channelID),
		checkEnvSecret("PAINTRESS_SLACK_APP_TOKEN", appToken),
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

const setupGuide = `
Setup Guide:

  PAINTRESS_SLACK_TOKEN — Bot User OAuth Token
    1. Go to https://api.slack.com/apps and click "Create New App"
    2. Choose "From scratch", give it a name, select workspace
    3. Go to "OAuth & Permissions" in the left sidebar
    4. Under "Scopes" > "Bot Token Scopes", add: chat:write
    5. Click "Install to Workspace" and authorize
    6. Copy the "Bot User OAuth Token" (starts with xoxb-)

  PAINTRESS_SLACK_CHANNEL_ID — Target channel ID
    1. Open Slack and right-click the target channel
    2. Select "View channel details"
    3. Scroll to the bottom — Channel ID is displayed there
    4. Invite the bot: /invite @YourBotName

  PAINTRESS_SLACK_APP_TOKEN — App-Level Token for Socket Mode
    1. In your app settings, go to "Socket Mode" and enable it
    2. Go to "Basic Information" > "App-Level Tokens"
    3. Click "Generate Token and Scopes"
    4. Add scope: connections:write
    5. Give it a name and click "Generate"
    6. Copy the token (starts with xapp-)

  Example:
    export PAINTRESS_SLACK_TOKEN="xoxb-123-456-abc..."
    export PAINTRESS_SLACK_CHANNEL_ID="C01234567"
    export PAINTRESS_SLACK_APP_TOKEN="xapp-1-A0123..."
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
			fmt.Fprintln(w, "paintress-slack doctor")
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
				fmt.Fprint(w, setupGuide)
				return fmt.Errorf("some checks failed")
			}
			fmt.Fprintln(w, "All checks passed.")
			return nil
		},
	}
}
