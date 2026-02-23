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
			cfg, err := parseNotifyConfig(
				os.Getenv("PAINTRESS_SLACK_TOKEN"),
				os.Getenv("PAINTRESS_SLACK_CHANNEL_ID"),
			)
			if err != nil {
				return err
			}
			api := slack.New(cfg.token)
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

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			sm := socketmode.New(api)
			go runSocketMode(ctx, sm, events)

			approved, err := sendApprove(ctx, api, cfg.channelID, args[0], timeout, events)
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
//
// sm.RunContext errors are captured: if RunContext exits with a non-nil error
// (and ctx is not cancelled), the error is surfaced through the events channel
// so sendApprove can return it instead of silently timing out.
//
// EventTypeConnectionError is treated as transient (Socket Mode retries with
// backoff internally). Only EventTypeInvalidAuth is terminal.
func runSocketMode(ctx context.Context, sm *socketmode.Client, events chan<- socketEvent) {
	runErr := make(chan error, 1)
	go func() {
		runErr <- sm.RunContext(ctx)
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-sm.Events:
			if !ok {
				// sm.Events closed — check if RunContext returned an error.
				select {
				case err := <-runErr:
					if err != nil && ctx.Err() == nil {
						select {
						case events <- socketEvent{Err: fmt.Errorf("socket mode: %w", err)}:
						case <-ctx.Done():
						}
					}
				default:
				}
				return
			}
			switch evt.Type {
			case socketmode.EventTypeInvalidAuth:
				select {
				case events <- socketEvent{Err: fmt.Errorf("invalid auth (check PAINTRESS_SLACK_APP_TOKEN)")}:
				case <-ctx.Done():
				}
				return
			case socketmode.EventTypeConnectionError:
				continue // transient — Socket Mode retries with backoff
			case socketmode.EventTypeInteractive:
				// handled below
			default:
				continue
			}
			cb, ok := evt.Data.(slack.InteractionCallback)
			if !ok {
				continue
			}
			sm.Ack(*evt.Request)

			for _, action := range cb.ActionCallback.BlockActions {
				select {
				case events <- socketEvent{
					ActionID:  action.ActionID,
					MessageTS: cb.Message.Timestamp,
				}:
				case <-ctx.Done():
					return
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
