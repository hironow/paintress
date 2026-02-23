package main

import (
	"fmt"
	"testing"

	"github.com/slack-go/slack"
)

// mockBot implements botAPI for testing.
type mockBot struct {
	postFunc   func(channelID string, opts ...slack.MsgOption) (string, string, error)
	updateFunc func(channelID, timestamp string, opts ...slack.MsgOption) (string, string, string, error)
}

func (m *mockBot) PostMessage(channelID string, opts ...slack.MsgOption) (string, string, error) {
	if m.postFunc != nil {
		return m.postFunc(channelID, opts...)
	}
	return "ch-1", "1234567890.123456", nil
}

func (m *mockBot) UpdateMessage(channelID, timestamp string, opts ...slack.MsgOption) (string, string, string, error) {
	if m.updateFunc != nil {
		return m.updateFunc(channelID, timestamp, opts...)
	}
	return "ch-1", "1234567890.123456", "", nil
}

// --- parseBotConfig tests ---

func TestParseBotConfig_Valid(t *testing.T) {
	// given
	token := "xoxb-123-456-abc"
	channelID := "C01234567"
	appToken := "xapp-1-abc-def"

	// when
	cfg, err := parseBotConfig(token, channelID, appToken)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.token != token {
		t.Errorf("token = %q, want %q", cfg.token, token)
	}
	if cfg.channelID != channelID {
		t.Errorf("channelID = %q, want %q", cfg.channelID, channelID)
	}
	if cfg.appToken != appToken {
		t.Errorf("appToken = %q, want %q", cfg.appToken, appToken)
	}
}

func TestParseBotConfig_MissingToken(t *testing.T) {
	_, err := parseBotConfig("", "C01234567", "xapp-1-abc")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestParseBotConfig_MissingChannelID(t *testing.T) {
	_, err := parseBotConfig("xoxb-123", "", "xapp-1-abc")
	if err == nil {
		t.Fatal("expected error for missing channel ID")
	}
}

func TestParseBotConfig_MissingAppToken(t *testing.T) {
	_, err := parseBotConfig("xoxb-123", "C01234567", "")
	if err == nil {
		t.Fatal("expected error for missing app token")
	}
}

// --- sendNotify tests ---

func TestSendNotify_Success(t *testing.T) {
	bot := &mockBot{}
	err := sendNotify(bot, "C01234567", "test message")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendNotify_Failure(t *testing.T) {
	bot := &mockBot{
		postFunc: func(string, ...slack.MsgOption) (string, string, error) {
			return "", "", fmt.Errorf("API error")
		},
	}
	err := sendNotify(bot, "C01234567", "test message")
	if err == nil {
		t.Fatal("expected error on send failure")
	}
}
