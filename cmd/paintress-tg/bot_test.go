package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// mockBot implements botAPI for testing.
type mockBot struct {
	sendFunc    func(c tgbotapi.Chattable) (tgbotapi.Message, error)
	requestFunc func(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	updatesCh   chan tgbotapi.Update
	stopped     bool
}

func (m *mockBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	if m.sendFunc != nil {
		return m.sendFunc(c)
	}
	return tgbotapi.Message{MessageID: 42}, nil
}

func (m *mockBot) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	if m.requestFunc != nil {
		return m.requestFunc(c)
	}
	return &tgbotapi.APIResponse{Ok: true}, nil
}

func (m *mockBot) GetUpdatesChan(_ tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return m.updatesCh
}

func (m *mockBot) StopReceivingUpdates() {
	m.stopped = true
}

// --- parseBotConfig tests ---

func TestParseBotConfig_Valid(t *testing.T) {
	// given
	token := "123456:ABC-DEF"
	chatID := "987654321"

	// when
	cfg, err := parseBotConfig(token, chatID)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.token != token {
		t.Errorf("token = %q, want %q", cfg.token, token)
	}
	if cfg.chatID != 987654321 {
		t.Errorf("chatID = %d, want 987654321", cfg.chatID)
	}
}

func TestParseBotConfig_MissingToken(t *testing.T) {
	// when
	_, err := parseBotConfig("", "123")

	// then
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestParseBotConfig_MissingChatID(t *testing.T) {
	// when
	_, err := parseBotConfig("token", "")

	// then
	if err == nil {
		t.Fatal("expected error for missing chat ID")
	}
}

func TestParseBotConfig_InvalidChatID(t *testing.T) {
	// when
	_, err := parseBotConfig("token", "not-a-number")

	// then
	if err == nil {
		t.Fatal("expected error for non-integer chat ID")
	}
}

// --- sendNotify tests ---

func TestSendNotify_Success(t *testing.T) {
	// given
	bot := &mockBot{}

	// when
	err := sendNotify(bot, 123, "test message")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendNotify_Failure(t *testing.T) {
	// given
	bot := &mockBot{
		sendFunc: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
			return tgbotapi.Message{}, fmt.Errorf("API error")
		},
	}

	// when
	err := sendNotify(bot, 123, "test message")

	// then
	if err == nil {
		t.Fatal("expected error on send failure")
	}
}

// --- sendApprove tests ---

func TestSendApprove_Approved(t *testing.T) {
	// given
	ch := make(chan tgbotapi.Update, 1)
	bot := &mockBot{updatesCh: ch}

	ch <- tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb1",
			Data: "approve",
			Message: &tgbotapi.Message{
				MessageID: 42,
			},
		},
	}

	// when
	approved, err := sendApprove(context.Background(), bot, 123, "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true")
	}
	if !bot.stopped {
		t.Error("expected StopReceivingUpdates to be called")
	}
}

func TestSendApprove_Denied(t *testing.T) {
	// given
	ch := make(chan tgbotapi.Update, 1)
	bot := &mockBot{updatesCh: ch}

	ch <- tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb2",
			Data: "deny",
			Message: &tgbotapi.Message{
				MessageID: 42,
			},
		},
	}

	// when
	approved, err := sendApprove(context.Background(), bot, 123, "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for deny")
	}
}

func TestSendApprove_Timeout(t *testing.T) {
	// given: no updates will arrive
	ch := make(chan tgbotapi.Update)
	bot := &mockBot{updatesCh: ch}

	// when: very short timeout
	approved, err := sendApprove(context.Background(), bot, 123, "approve?", 50*time.Millisecond)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false on timeout")
	}
}

func TestSendApprove_SendFailure(t *testing.T) {
	// given
	bot := &mockBot{
		sendFunc: func(_ tgbotapi.Chattable) (tgbotapi.Message, error) {
			return tgbotapi.Message{}, fmt.Errorf("API error")
		},
		updatesCh: make(chan tgbotapi.Update),
	}

	// when
	_, err := sendApprove(context.Background(), bot, 123, "approve?", 5*time.Second)

	// then
	if err == nil {
		t.Fatal("expected error on send failure")
	}
}

func TestSendApprove_IgnoresUnrelatedCallback(t *testing.T) {
	// given: callback for a different message, then the correct one
	ch := make(chan tgbotapi.Update, 2)
	bot := &mockBot{updatesCh: ch}

	// Unrelated callback (different message ID)
	ch <- tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "unrelated",
			Data: "deny",
			Message: &tgbotapi.Message{
				MessageID: 999, // different from our sent message (42)
			},
		},
	}
	// Our callback
	ch <- tgbotapi.Update{
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "ours",
			Data: "approve",
			Message: &tgbotapi.Message{
				MessageID: 42,
			},
		},
	}

	// when
	approved, err := sendApprove(context.Background(), bot, 123, "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true (should skip unrelated callback)")
	}
}

func TestSendApprove_ContextCancel(t *testing.T) {
	// given
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	ch := make(chan tgbotapi.Update)
	bot := &mockBot{updatesCh: ch}

	// when
	approved, err := sendApprove(ctx, bot, 123, "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false on context cancel")
	}
}
