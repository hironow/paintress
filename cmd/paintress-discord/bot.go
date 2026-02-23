package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/bwmarrin/discordgo"
)

// botAPI abstracts the subset of Discord API used by this binary.
type botAPI interface {
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	AddHandler(handler interface{}) func()
	Open() error
	Close() error
}

// botConfig holds validated configuration for bot operations.
type botConfig struct {
	token     string
	channelID string
}

// parseBotConfig reads and validates environment-sourced configuration.
func parseBotConfig(token, channelID string) (botConfig, error) {
	if token == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_DISCORD_TOKEN is required")
	}
	if channelID == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_DISCORD_CHANNEL_ID is required")
	}
	return botConfig{token: token, channelID: channelID}, nil
}

// sendNotify sends a text message to the configured channel and returns immediately.
// Does NOT require Open() — uses REST API only.
func sendNotify(bot botAPI, channelID, message string) error {
	_, err := bot.ChannelMessageSend(channelID, message)
	return err
}

// sendApprove sends a message with Approve/Deny buttons and waits for an
// InteractionCreate event via WebSocket Gateway. Returns true if approved,
// false if denied or timed out.
func sendApprove(ctx context.Context, bot botAPI, channelID, message string, timeout time.Duration) (bool, error) {
	if err := bot.Open(); err != nil {
		return false, fmt.Errorf("failed to open Discord session: %w", err)
	}
	defer bot.Close()

	result := make(chan bool, 1)
	var sentID atomic.Value // stores the sent message ID (string)

	// Register handler BEFORE sending the message to avoid dropping
	// fast button clicks that arrive between send and registration.
	removeHandler := bot.AddHandler(func(_ *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionMessageComponent {
			return
		}
		id, _ := sentID.Load().(string)
		if id == "" || i.Message == nil || i.Message.ID != id {
			return
		}

		data := i.MessageComponentData()
		// Non-blocking send: if the result channel already has a value
		// (e.g. duplicate clicks), drop silently to avoid goroutine leaks.
		switch data.CustomID {
		case "approve":
			select {
			case result <- true:
			default:
			}
		case "deny":
			select {
			case result <- false:
			default:
			}
		}
	})
	defer removeHandler()

	sent, err := bot.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Content: message,
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					discordgo.Button{
						Label:    "Approve",
						Style:    discordgo.SuccessButton,
						CustomID: "approve",
					},
					discordgo.Button{
						Label:    "Deny",
						Style:    discordgo.DangerButton,
						CustomID: "deny",
					},
				},
			},
		},
	})
	if err != nil {
		return false, fmt.Errorf("failed to send approval message: %w", err)
	}
	sentID.Store(sent.ID)

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		return false, nil
	case approved := <-result:
		return approved, nil
	}
}
