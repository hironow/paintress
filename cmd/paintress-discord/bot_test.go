package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// mockBot implements botAPI for testing.
type mockBot struct {
	sendFunc        func(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	sendComplexFunc func(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	openFunc        func() error
	closed          bool
	interactionCh   chan *discordgo.InteractionCreate
}

func (m *mockBot) ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if m.sendFunc != nil {
		return m.sendFunc(channelID, content, options...)
	}
	return &discordgo.Message{ID: "msg-42"}, nil
}

func (m *mockBot) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	if m.sendComplexFunc != nil {
		return m.sendComplexFunc(channelID, data, options...)
	}
	return &discordgo.Message{ID: "msg-42"}, nil
}

func (m *mockBot) AddHandler(handler interface{}) func() {
	fn, ok := handler.(func(*discordgo.Session, *discordgo.InteractionCreate))
	if !ok {
		return func() {}
	}
	go func() {
		for i := range m.interactionCh {
			fn(nil, i)
		}
	}()
	return func() {}
}

func (m *mockBot) Open() error {
	if m.openFunc != nil {
		return m.openFunc()
	}
	return nil
}

func (m *mockBot) Close() error {
	m.closed = true
	return nil
}

// --- parseBotConfig tests ---

func TestParseBotConfig_Valid(t *testing.T) {
	// given
	token := "Bot MTIz.abc.xyz"
	channelID := "123456789"

	// when
	cfg, err := parseBotConfig(token, channelID)

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

func TestParseBotConfig_MissingToken(t *testing.T) {
	// when
	_, err := parseBotConfig("", "123")

	// then
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestParseBotConfig_MissingChannelID(t *testing.T) {
	// when
	_, err := parseBotConfig("token", "")

	// then
	if err == nil {
		t.Fatal("expected error for missing channel ID")
	}
}

// --- sendNotify tests ---

func TestSendNotify_Success(t *testing.T) {
	// given
	bot := &mockBot{}

	// when
	err := sendNotify(bot, "ch-1", "test message")

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendNotify_Failure(t *testing.T) {
	// given
	bot := &mockBot{
		sendFunc: func(string, string, ...discordgo.RequestOption) (*discordgo.Message, error) {
			return nil, fmt.Errorf("API error")
		},
	}

	// when
	err := sendNotify(bot, "ch-1", "test message")

	// then
	if err == nil {
		t.Fatal("expected error on send failure")
	}
}

// --- sendApprove tests ---

func TestSendApprove_Approved(t *testing.T) {
	// given
	ch := make(chan *discordgo.InteractionCreate, 1)
	bot := &mockBot{interactionCh: ch}

	ch <- &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Message: &discordgo.Message{
				ID: "msg-42",
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "approve",
			},
		},
	}

	// when
	approved, err := sendApprove(context.Background(), bot, "ch-1", "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true")
	}
	if !bot.closed {
		t.Error("expected Close to be called")
	}
}

func TestSendApprove_Denied(t *testing.T) {
	// given
	ch := make(chan *discordgo.InteractionCreate, 1)
	bot := &mockBot{interactionCh: ch}

	ch <- &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Message: &discordgo.Message{
				ID: "msg-42",
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "deny",
			},
		},
	}

	// when
	approved, err := sendApprove(context.Background(), bot, "ch-1", "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false for deny")
	}
}

func TestSendApprove_Timeout(t *testing.T) {
	// given: no interactions will arrive
	ch := make(chan *discordgo.InteractionCreate)
	bot := &mockBot{interactionCh: ch}

	// when: very short timeout
	approved, err := sendApprove(context.Background(), bot, "ch-1", "approve?", 50*time.Millisecond)

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
		sendComplexFunc: func(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
			return nil, fmt.Errorf("API error")
		},
		interactionCh: make(chan *discordgo.InteractionCreate),
	}

	// when
	_, err := sendApprove(context.Background(), bot, "ch-1", "approve?", 5*time.Second)

	// then
	if err == nil {
		t.Fatal("expected error on send failure")
	}
}

func TestSendApprove_OpenFailure(t *testing.T) {
	// given
	bot := &mockBot{
		openFunc: func() error {
			return fmt.Errorf("connection failed")
		},
	}

	// when
	_, err := sendApprove(context.Background(), bot, "ch-1", "approve?", 5*time.Second)

	// then
	if err == nil {
		t.Fatal("expected error on open failure")
	}
}

func TestSendApprove_IgnoresUnrelatedInteraction(t *testing.T) {
	// given: interaction for a different message, then the correct one
	ch := make(chan *discordgo.InteractionCreate, 2)
	bot := &mockBot{interactionCh: ch}

	// Unrelated interaction (different message ID)
	ch <- &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Message: &discordgo.Message{
				ID: "msg-999", // different from our sent message
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "deny",
			},
		},
	}
	// Our interaction
	ch <- &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionMessageComponent,
			Message: &discordgo.Message{
				ID: "msg-42",
			},
			Data: discordgo.MessageComponentInteractionData{
				CustomID: "approve",
			},
		},
	}

	// when
	approved, err := sendApprove(context.Background(), bot, "ch-1", "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !approved {
		t.Error("expected approved=true (should skip unrelated interaction)")
	}
}

func TestSendApprove_ContextCancel(t *testing.T) {
	// given
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled

	ch := make(chan *discordgo.InteractionCreate)
	bot := &mockBot{interactionCh: ch}

	// when
	approved, err := sendApprove(ctx, bot, "ch-1", "approve?", 5*time.Second)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if approved {
		t.Error("expected approved=false on context cancel")
	}
}
