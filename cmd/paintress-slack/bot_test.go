package main

import (
	"context"
	"fmt"
	"testing"
	"time"

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

// --- parseNotifyConfig tests ---

func TestParseNotifyConfig_Valid(t *testing.T) {
	// given
	token := "xoxb-123-456-abc"
	channelID := "C01234567"

	// when
	cfg, err := parseNotifyConfig(token, channelID)

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
}

func TestParseNotifyConfig_MissingToken(t *testing.T) {
	_, err := parseNotifyConfig("", "C01234567")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestParseNotifyConfig_MissingChannelID(t *testing.T) {
	_, err := parseNotifyConfig("xoxb-123", "")
	if err == nil {
		t.Fatal("expected error for missing channel ID")
	}
}

func TestParseNotifyConfig_DoesNotRequireAppToken(t *testing.T) {
	// given: only token and channelID (no app token)
	cfg, err := parseNotifyConfig("xoxb-123", "C01234567")

	// then: succeeds without app token
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.appToken != "" {
		t.Errorf("appToken should be empty, got %q", cfg.appToken)
	}
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

// --- sendApprove tests ---

func TestSendApprove_Approved(t *testing.T) {
	ch := make(chan socketEvent, 1)
	bot := &mockBot{}
	ch <- socketEvent{ActionID: "approve", MessageTS: "1234567890.123456"}

	approved, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true")
	}
}

func TestSendApprove_Denied(t *testing.T) {
	ch := make(chan socketEvent, 1)
	bot := &mockBot{}
	ch <- socketEvent{ActionID: "deny", MessageTS: "1234567890.123456"}

	approved, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for deny")
	}
}

func TestSendApprove_Timeout(t *testing.T) {
	ch := make(chan socketEvent) // no events
	bot := &mockBot{}

	approved, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 50*time.Millisecond, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false on timeout")
	}
}

func TestSendApprove_SendFailure(t *testing.T) {
	ch := make(chan socketEvent)
	bot := &mockBot{
		postFunc: func(string, ...slack.MsgOption) (string, string, error) {
			return "", "", fmt.Errorf("API error")
		},
	}

	_, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)
	if err == nil {
		t.Fatal("expected error on send failure")
	}
}

func TestSendApprove_IgnoresUnrelated(t *testing.T) {
	ch := make(chan socketEvent, 2)
	bot := &mockBot{}

	// Unrelated action (different timestamp)
	ch <- socketEvent{ActionID: "approve", MessageTS: "9999999999.999999"}
	// Our action
	ch <- socketEvent{ActionID: "approve", MessageTS: "1234567890.123456"}

	approved, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true (should skip unrelated)")
	}
}

func TestSendApprove_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch := make(chan socketEvent)
	bot := &mockBot{}

	approved, err := sendApprove(ctx, bot, "C01234567", "approve?", 5*time.Second, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false on context cancel")
	}
}

func TestSendApprove_DuplicateClicks(t *testing.T) {
	ch := make(chan socketEvent, 3)
	bot := &mockBot{}

	ch <- socketEvent{ActionID: "approve", MessageTS: "1234567890.123456"}
	ch <- socketEvent{ActionID: "approve", MessageTS: "1234567890.123456"} // dup
	ch <- socketEvent{ActionID: "deny", MessageTS: "1234567890.123456"}    // late

	approved, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true (first click wins)")
	}
}

func TestSendApprove_SocketModeError(t *testing.T) {
	// given: a socketEvent with an error (e.g. invalid auth)
	ch := make(chan socketEvent, 1)
	bot := &mockBot{}
	ch <- socketEvent{Err: fmt.Errorf("socket mode: invalid auth")}

	// when
	approved, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)

	// then: error is surfaced (not treated as denial)
	if err == nil {
		t.Fatal("expected error for socket mode failure")
	}
	if approved {
		t.Error("expected approved=false on socket mode error")
	}
}

func TestSendApprove_SocketModeErrorBeforeInteraction(t *testing.T) {
	// given: error arrives before any interactive event
	ch := make(chan socketEvent, 2)
	bot := &mockBot{}
	ch <- socketEvent{Err: fmt.Errorf("socket mode: connection error")}
	ch <- socketEvent{ActionID: "approve", MessageTS: "1234567890.123456"}

	// when
	_, err := sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)

	// then: error takes priority, approval never reached
	if err == nil {
		t.Fatal("expected error for socket mode failure")
	}
}

func TestSendApprove_UpdatesMessage(t *testing.T) {
	ch := make(chan socketEvent, 1)
	var updatedTS string
	bot := &mockBot{
		updateFunc: func(_ string, ts string, _ ...slack.MsgOption) (string, string, string, error) {
			updatedTS = ts
			return "", ts, "", nil
		},
	}
	ch <- socketEvent{ActionID: "approve", MessageTS: "1234567890.123456"}

	sendApprove(context.Background(), bot, "C01234567", "approve?", 5*time.Second, ch)

	if updatedTS != "1234567890.123456" {
		t.Errorf("expected message update with ts=%q, got %q", "1234567890.123456", updatedTS)
	}
}
