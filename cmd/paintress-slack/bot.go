package main

import (
	"fmt"

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

// parseBotConfig reads and validates environment-sourced configuration.
func parseBotConfig(token, channelID, appToken string) (botConfig, error) {
	if token == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_SLACK_TOKEN is required")
	}
	if channelID == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_SLACK_CHANNEL_ID is required")
	}
	if appToken == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_SLACK_APP_TOKEN is required")
	}
	return botConfig{token: token, channelID: channelID, appToken: appToken}, nil
}

// sendNotify sends a text message to the configured channel and returns immediately.
func sendNotify(bot botAPI, channelID, message string) error {
	_, _, err := bot.PostMessage(channelID, slack.MsgOptionText(message, false))
	return err
}
