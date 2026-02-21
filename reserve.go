package paintress

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ReserveParty manages model fallback when the primary model hits rate limits.
// In the game, when the active party falls, reserve members step in.
// Here, when Opus hits rate limits, Sonnet takes over.
//
// The monitoring goroutine watches for rate-limit signals in Claude's output
// and switches models automatically.
type ReserveParty struct {
	mu sync.RWMutex

	primary string   // e.g. "opus"
	reserve []string // e.g. ["sonnet", "haiku"]
	active  string   // currently active model
	logger  *Logger

	// Rate limit tracking
	rateLimitHits int
	lastHitTime   time.Time
	cooldownUntil time.Time
}

func NewReserveParty(primary string, reserves []string, logger *Logger) *ReserveParty {
	return &ReserveParty{
		primary: primary,
		reserve: reserves,
		active:  primary,
		logger:  logger,
	}
}

// ActiveModel returns the currently active model.
func (rp *ReserveParty) ActiveModel() string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.active
}

// CheckOutput scans Claude's output for rate limit signals.
// Called from the output streaming goroutine.
// Returns true if a rate limit was detected.
func (rp *ReserveParty) CheckOutput(chunk string) bool {
	lower := strings.ToLower(chunk)
	rateLimitSignals := []string{
		"rate limit",
		"rate_limit",
		"too many requests",
		"quota exceeded",
		"usage limit",
		"try again later",
		"at capacity",
	}

	for _, signal := range rateLimitSignals {
		if strings.Contains(lower, signal) {
			rp.onRateLimitDetected()
			return true
		}
	}
	if hasHTTP429(lower) {
		rp.onRateLimitDetected()
		return true
	}
	return false
}

func hasHTTP429(text string) bool {
	for i := 0; i+3 <= len(text); i++ {
		if text[i:i+3] != "429" {
			continue
		}
		prevOK := i == 0 || !isAlphaNum(text[i-1])
		nextOK := i+3 == len(text) || !isAlphaNum(text[i+3])
		if prevOK && nextOK {
			return true
		}
	}
	return false
}

func isAlphaNum(b byte) bool {
	if b >= '0' && b <= '9' {
		return true
	}
	if b >= 'a' && b <= 'z' {
		return true
	}
	if b >= 'A' && b <= 'Z' {
		return true
	}
	return false
}

func (rp *ReserveParty) onRateLimitDetected() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.rateLimitHits++
	rp.lastHitTime = time.Now()
	// Cooldown: wait 30 minutes before trying primary again
	rp.cooldownUntil = time.Now().Add(30 * time.Minute)

	if rp.active == rp.primary && len(rp.reserve) > 0 {
		prev := rp.active
		rp.active = rp.reserve[0]
		rp.logger.Warn("%s", fmt.Sprintf(Msg("reserve_activated"), prev, rp.active))
	}
}

// TryRecoverPrimary checks if cooldown has passed and switches back to primary.
// Called before each expedition.
func (rp *ReserveParty) TryRecoverPrimary() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.active != rp.primary && time.Now().After(rp.cooldownUntil) {
		prev := rp.active
		rp.active = rp.primary
		rp.logger.OK("%s", fmt.Sprintf(Msg("primary_recovered"), prev, rp.active))
	}
}

// ForceReserve manually switches to reserve (e.g., after a timeout that looks rate-limit-related).
func (rp *ReserveParty) ForceReserve() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.active == rp.primary && len(rp.reserve) > 0 {
		prev := rp.active
		rp.active = rp.reserve[0]
		rp.cooldownUntil = time.Now().Add(30 * time.Minute)
		rp.logger.Warn("%s", fmt.Sprintf(Msg("reserve_forced"), prev, rp.active))
	}
}

// Status returns a human-readable status string.
func (rp *ReserveParty) Status() string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	if rp.active == rp.primary {
		return fmt.Sprintf("Active: %s (primary) | Reserve: %v | Hits: %d",
			rp.active, rp.reserve, rp.rateLimitHits)
	}

	remaining := time.Until(rp.cooldownUntil).Round(time.Minute)
	return fmt.Sprintf("Active: %s (RESERVE) | Primary %s recovering (%v) | Hits: %d",
		rp.active, rp.primary, remaining, rp.rateLimitHits)
}

// FormatForPrompt provides model context for the prompt.
func (rp *ReserveParty) FormatForPrompt() string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	if rp.active == rp.primary {
		return fmt.Sprintf("Model: %s (primary)", rp.active)
	}
	return fmt.Sprintf("Model: %s (reserve — primary %s is recovering from rate limit)", rp.active, rp.primary)
}
