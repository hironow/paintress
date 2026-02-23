package main

import (
	"context"
	"fmt"
	"time"

	"github.com/slack-go/slack"
)

// botAPI abstracts the subset of Slack API used by this binary.
type botAPI interface {
	PostMessage(channelID string, opts ...slack.MsgOption) (string, string, error)
	UpdateMessage(channelID, timestamp string, opts ...slack.MsgOption) (string, string, string, error)
}

// botConfig holds validated configuration for bot operations.
type botConfig struct {
	token     string
	channelID string
	appToken  string
}

// parseNotifyConfig validates config for notify (token + channelID only).
func parseNotifyConfig(token, channelID string) (botConfig, error) {
	if token == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_SLACK_TOKEN is required")
	}
	if channelID == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_SLACK_CHANNEL_ID is required")
	}
	return botConfig{token: token, channelID: channelID}, nil
}

// parseBotConfig validates config for approve (token + channelID + appToken).
func parseBotConfig(token, channelID, appToken string) (botConfig, error) {
	cfg, err := parseNotifyConfig(token, channelID)
	if err != nil {
		return botConfig{}, err
	}
	if appToken == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_SLACK_APP_TOKEN is required")
	}
	cfg.appToken = appToken
	return cfg, nil
}

// socketEvent represents an incoming interactive action from Socket Mode.
// When Err is non-nil, it signals a connection/auth failure instead of a user action.
type socketEvent struct {
	ActionID  string
	MessageTS string
	Err       error
}

// sendNotify sends a text message to the configured channel and returns immediately.
func sendNotify(bot botAPI, channelID, message string) error {
	_, _, err := bot.PostMessage(channelID, slack.MsgOptionText(message, false))
	return err
}

// sendApprove sends a message with Approve/Deny buttons and waits for a
// button click via the events channel. Returns true if approved, false if
// denied or timed out.
func sendApprove(ctx context.Context, bot botAPI, channelID, message string, timeout time.Duration, events <-chan socketEvent) (bool, error) {
	approveBtn := slack.NewButtonBlockElement("approve", "approve", slack.NewTextBlockObject(slack.PlainTextType, "Approve", false, false))
	approveBtn.Style = slack.StylePrimary

	denyBtn := slack.NewButtonBlockElement("deny", "deny", slack.NewTextBlockObject(slack.PlainTextType, "Deny", false, false))
	denyBtn.Style = slack.StyleDanger

	actionBlock := slack.NewActionBlock("approval_actions", approveBtn, denyBtn)
	textBlock := slack.NewSectionBlock(slack.NewTextBlockObject(slack.MarkdownType, message, false, false), nil, nil)

	_, sentTS, err := bot.PostMessage(channelID, slack.MsgOptionBlocks(textBlock, actionBlock))
	if err != nil {
		return false, fmt.Errorf("failed to send approval message: %w", err)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return false, nil
		case ev := <-events:
			if ev.Err != nil {
				return false, fmt.Errorf("socket mode failure: %w", ev.Err)
			}
			if ev.MessageTS != sentTS {
				continue
			}

			var label string
			var approved bool
			switch ev.ActionID {
			case "approve":
				label = "Approved"
				approved = true
			case "deny":
				label = "Denied"
			default:
				continue
			}

			resultText := slack.NewSectionBlock(
				slack.NewTextBlockObject(slack.MarkdownType, message+"\n\n*"+label+"*", false, false),
				nil, nil,
			)
			_, _, _, _ = bot.UpdateMessage(channelID, sentTS, slack.MsgOptionBlocks(resultText))

			return approved, nil
		}
	}
}
