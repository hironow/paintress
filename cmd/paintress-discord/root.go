package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spf13/cobra"
)

// newBotFromEnv creates a real Discord session from environment variables.
// The session is NOT opened — caller decides whether Open() is needed.
func newBotFromEnv() (botAPI, botConfig, error) {
	cfg, err := parseBotConfig(
		os.Getenv("PAINTRESS_DISCORD_TOKEN"),
		os.Getenv("PAINTRESS_DISCORD_CHANNEL_ID"),
	)
	if err != nil {
		return nil, botConfig{}, err
	}
	session, err := discordgo.New("Bot " + cfg.token)
	if err != nil {
		return nil, botConfig{}, fmt.Errorf("failed to create Discord session: %w", err)
	}
	// InteractionCreate events are always delivered regardless of intents.
	// Request no intents to minimize Gateway traffic (least privilege).
	session.Identify.Intents = 0
	return session, cfg, nil
}

// NewRootCommand creates the root cobra command for paintress-discord.
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "paintress-discord",
		Short: "Discord companion for paintress notify/approve",
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
			bot, cfg, err := newBotFromEnv()
			if err != nil {
				return err
			}
			return sendNotify(bot, cfg.channelID, args[0])
		},
	}
}

func newApproveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "approve <message>",
		Short: "Send an approval request with Approve/Deny buttons",
		Long: `Send a message with Approve/Deny buttons and wait for a response.
Exit 0 = approved, exit 1 = denied or timed out.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			bot, cfg, err := newBotFromEnv()
			if err != nil {
				return err
			}
			timeout, _ := cmd.Flags().GetDuration("timeout")
			approved, err := sendApprove(cmd.Context(), bot, cfg.channelID, args[0], timeout)
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

// exitError wraps an exit code for the main function to handle.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit %d", e.code)
}
