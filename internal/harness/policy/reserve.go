package policy

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hironow/paintress/internal/domain"
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
	logger  domain.Logger

	// Rate limit tracking
	rateLimitHits int
	lastHitTime   time.Time
	cooldowns     map[string]time.Time // per-model cooldown expiry
}

func NewReserveParty(primary string, reserves []string, logger domain.Logger) *ReserveParty {
	if logger == nil {
		logger = &domain.NopLogger{}
	}
	return &ReserveParty{
		primary:   primary,
		reserve:   reserves,
		active:    primary,
		logger:    logger,
		cooldowns: make(map[string]time.Time),
	}
}

// ActiveModel returns the currently active model.
func (rp *ReserveParty) ActiveModel() string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.active
}

// IsOnReserve returns true if the active model is not the primary.
func (rp *ReserveParty) IsOnReserve() bool {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return rp.active != rp.primary
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

// nextAvailableModel returns the first model from the reserve list that is not
// in cooldown. Returns "" if all reserves are in cooldown.
// Caller must hold rp.mu.
func (rp *ReserveParty) nextAvailableModel() string {
	now := time.Now()
	// Check primary first
	if until, ok := rp.cooldowns[rp.primary]; !ok || now.After(until) {
		if rp.active != rp.primary {
			return rp.primary
		}
	}
	// Then check reserves in order
	for _, m := range rp.reserve {
		if m == rp.active {
			continue
		}
		if until, ok := rp.cooldowns[m]; !ok || now.After(until) {
			return m
		}
	}
	return ""
}

func (rp *ReserveParty) onRateLimitDetected() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.rateLimitHits++
	rp.lastHitTime = time.Now()
	// Put current model in cooldown
	rp.cooldowns[rp.active] = time.Now().Add(30 * time.Minute)

	if next := rp.nextAvailableModel(); next != "" {
		prev := rp.active
		rp.active = next
		rp.logger.Warn("%s", fmt.Sprintf(domain.Msg("reserve_activated"), prev, rp.active))
	}
}

// TryRecoverPrimary checks if primary's cooldown has passed and switches back.
// Called before each expedition.
func (rp *ReserveParty) TryRecoverPrimary() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	if rp.active == rp.primary {
		return
	}
	if until, ok := rp.cooldowns[rp.primary]; ok && !time.Now().After(until) {
		return
	}
	prev := rp.active
	rp.active = rp.primary
	rp.logger.OK("%s", fmt.Sprintf(domain.Msg("primary_recovered"), prev, rp.active))
}

// ForceReserve manually switches to the next available reserve model.
// Cascades through reserves if multiple are needed.
func (rp *ReserveParty) ForceReserve() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.cooldowns[rp.active] = time.Now().Add(30 * time.Minute)
	if next := rp.nextAvailableModel(); next != "" {
		prev := rp.active
		rp.active = next
		rp.logger.Warn("%s", fmt.Sprintf(domain.Msg("reserve_forced"), prev, rp.active))
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

	// Show primary cooldown remaining
	now := time.Now()
	var parts []string
	if until, ok := rp.cooldowns[rp.primary]; ok && now.Before(until) {
		parts = append(parts, fmt.Sprintf("%s recovering (%v)", rp.primary, time.Until(until).Round(time.Minute)))
	}
	for _, m := range rp.reserve {
		if until, ok := rp.cooldowns[m]; ok && now.Before(until) {
			parts = append(parts, fmt.Sprintf("%s recovering (%v)", m, time.Until(until).Round(time.Minute)))
		}
	}
	cooldownInfo := strings.Join(parts, ", ")
	if cooldownInfo == "" {
		cooldownInfo = "none in cooldown"
	}
	return fmt.Sprintf("Active: %s (RESERVE) | %s | Hits: %d",
		rp.active, cooldownInfo, rp.rateLimitHits)
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
