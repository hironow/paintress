package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// botAPI abstracts the subset of Telegram Bot API used by this binary.
type botAPI interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	StopReceivingUpdates()
}

// botConfig holds validated configuration for bot operations.
type botConfig struct {
	token  string
	chatID int64
}

// parseBotConfig reads and validates environment-sourced configuration.
func parseBotConfig(token, chatIDStr string) (botConfig, error) {
	if token == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_TG_TOKEN is required")
	}
	if chatIDStr == "" {
		return botConfig{}, fmt.Errorf("PAINTRESS_TG_CHAT_ID is required")
	}
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		return botConfig{}, fmt.Errorf("PAINTRESS_TG_CHAT_ID must be an integer: %w", err)
	}
	return botConfig{token: token, chatID: chatID}, nil
}

// sendNotify sends a text message to the configured chat and returns immediately.
func sendNotify(bot botAPI, chatID int64, message string) error {
	msg := tgbotapi.NewMessage(chatID, message)
	_, err := bot.Send(msg)
	return err
}

// sendApprove sends a message with Approve/Deny inline buttons and waits
// for a CallbackQuery response. Returns true if approved, false if denied
// or timed out.
func sendApprove(ctx context.Context, bot botAPI, chatID int64, message string, timeout time.Duration) (bool, error) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Approve", "approve"),
			tgbotapi.NewInlineKeyboardButtonData("Deny", "deny"),
		),
	)
	msg := tgbotapi.NewMessage(chatID, message)
	msg.ReplyMarkup = keyboard

	sent, err := bot.Send(msg)
	if err != nil {
		return false, fmt.Errorf("failed to send approval message: %w", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30 // long-poll interval (seconds)
	updates := bot.GetUpdatesChan(u)
	defer bot.StopReceivingUpdates()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-timeoutCtx.Done():
			return false, nil
		case update, ok := <-updates:
			if !ok {
				return false, nil
			}
			if update.CallbackQuery == nil {
				continue
			}
			// Only respond to callbacks on our specific message
			if update.CallbackQuery.Message == nil || update.CallbackQuery.Message.MessageID != sent.MessageID {
				continue
			}

			// Acknowledge the callback (removes loading spinner)
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			_, _ = bot.Request(callback)

			switch update.CallbackQuery.Data {
			case "approve":
				return true, nil
			case "deny":
				return false, nil
			}
		}
	}
}
