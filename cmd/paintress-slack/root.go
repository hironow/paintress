package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
	"github.com/spf13/cobra"
)

// newBotFromEnv creates a Slack API client from environment variables.
func newBotFromEnv() (*slack.Client, botConfig, error) {
	cfg, err := parseBotConfig(
		os.Getenv("PAINTRESS_SLACK_TOKEN"),
		os.Getenv("PAINTRESS_SLACK_CHANNEL_ID"),
		os.Getenv("PAINTRESS_SLACK_APP_TOKEN"),
	)
	if err != nil {
		return nil, botConfig{}, err
	}
	api := slack.New(cfg.token, slack.OptionAppLevelToken(cfg.appToken))
	return api, cfg, nil
}

// NewRootCommand creates the root cobra command for paintress-slack.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "paintress-slack",
		Short: "Slack companion for paintress notify/approve",
	}

	rootCmd.PersistentFlags().Duration("timeout", 5*time.Minute, "Timeout for approval wait")

	rootCmd.AddCommand(
		newNotifyCommand(),
		newApproveCommand(),
		newDoctorCommand(),
	)

	return rootCmd
}

func newNotifyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "notify <message>",
		Short: "Send a notification message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, cfg, err := newBotFromEnv()
			if err != nil {
				return err
			}
			return sendNotify(api, cfg.channelID, args[0])
		},
	}
}

func newApproveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <message>",
		Short: "Send an approval request with Approve/Deny buttons",
		Long: `Send a message with Approve/Deny buttons and wait for a response via Socket Mode.
Exit 0 = approved, exit 1 = denied or timed out.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			api, cfg, err := newBotFromEnv()
			if err != nil {
				return err
			}

			timeout, _ := cmd.Flags().GetDuration("timeout")
			events := make(chan socketEvent, 1)

			sm := socketmode.New(api)
			go runSocketMode(cmd.Context(), sm, events)

			approved, err := sendApprove(cmd.Context(), api, cfg.channelID, args[0], timeout, events)
			if err != nil {
				return err
			}
			if !approved {
				return &exitError{code: 1}
			}
			return nil
		},
	}
}

// runSocketMode runs the Socket Mode client, converting interactive events
// to socketEvent structs on the events channel. Blocks until ctx is cancelled.
func runSocketMode(ctx context.Context, sm *socketmode.Client, events chan<- socketEvent) {
	go sm.RunContext(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-sm.Events:
			if !ok {
				return
			}
			if evt.Type != socketmode.EventTypeInteractive {
				continue
			}
			cb, ok := evt.Data.(slack.InteractionCallback)
			if !ok {
				continue
			}
			sm.Ack(*evt.Request)

			for _, action := range cb.ActionCallback.BlockActions {
				events <- socketEvent{
					ActionID:  action.ActionID,
					MessageTS: cb.Message.Timestamp,
				}
			}
		}
	}
}

// exitError wraps an exit code for the main function to handle.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}
