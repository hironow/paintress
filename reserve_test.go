package paintress

import (
	"io"
	"sync"
	"testing"
	"time"
)

func TestReserve_Status_Primary(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	s := rp.Status()
	if !containsStr(s, "primary") {
		t.Errorf("status should say primary: %q", s)
	}
	if !containsStr(s, "opus") {
		t.Errorf("status should mention opus: %q", s)
	}
	if !containsStr(s, "Hits: 0") {
		t.Errorf("status should show 0 hits: %q", s)
	}
}

func TestReserve_Status_AfterSwitch(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.CheckOutput("rate limit exceeded")
	s := rp.Status()
	if !containsStr(s, "RESERVE") {
		t.Errorf("status should say RESERVE: %q", s)
	}
	if !containsStr(s, "sonnet") {
		t.Errorf("status should mention sonnet: %q", s)
	}
	if !containsStr(s, "recovering") {
		t.Errorf("status should mention recovery: %q", s)
	}
	if !containsStr(s, "Hits: 1") {
		t.Errorf("status should show 1 hit: %q", s)
	}
}

func TestReserve_FormatForPrompt_Primary(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	s := rp.FormatForPrompt()
	if !containsStr(s, "opus") {
		t.Errorf("should mention opus: %q", s)
	}
	if !containsStr(s, "primary") {
		t.Errorf("should say primary: %q", s)
	}
}

func TestReserve_FormatForPrompt_Reserve(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.CheckOutput("429 too many requests")
	s := rp.FormatForPrompt()
	if !containsStr(s, "sonnet") {
		t.Errorf("should mention sonnet: %q", s)
	}
	if !containsStr(s, "reserve") {
		t.Errorf("should say reserve: %q", s)
	}
	if !containsStr(s, "rate limit") {
		t.Errorf("should mention rate limit: %q", s)
	}
}

func TestReserve_TryRecoverPrimary_BeforeCooldown(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.CheckOutput("rate limit")
	if rp.ActiveModel() != "sonnet" {
		t.Fatalf("should be on sonnet, got %q", rp.ActiveModel())
	}

	// Cooldown is 30 min — should NOT recover
	rp.TryRecoverPrimary()
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should stay on sonnet before cooldown: %q", rp.ActiveModel())
	}
}

func TestReserve_TryRecoverPrimary_AfterCooldown(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.CheckOutput("rate limit")

	// Manually set cooldown to past
	rp.mu.Lock()
	rp.cooldownUntil = time.Now().Add(-1 * time.Minute)
	rp.mu.Unlock()

	rp.TryRecoverPrimary()
	if rp.ActiveModel() != "opus" {
		t.Errorf("should recover to opus after cooldown: %q", rp.ActiveModel())
	}
}

func TestReserve_TryRecoverPrimary_AlreadyOnPrimary(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.TryRecoverPrimary() // No-op when already on primary
	if rp.ActiveModel() != "opus" {
		t.Errorf("should stay on opus: %q", rp.ActiveModel())
	}
}

func TestReserve_MultipleRateLimits(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"}, NewLogger(io.Discard, false))
	rp.CheckOutput("rate limit")
	rp.CheckOutput("429 again")

	// Should stay on first reserve (sonnet), not cascade to haiku
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should stay on sonnet: %q", rp.ActiveModel())
	}

	// Hit count should increase
	rp.mu.RLock()
	hits := rp.rateLimitHits
	rp.mu.RUnlock()
	if hits != 2 {
		t.Errorf("rateLimitHits = %d, want 2", hits)
	}
}

func TestReserve_ForceReserve_NoReserves(t *testing.T) {
	rp := NewReserveParty("opus", nil, NewLogger(io.Discard, false))
	rp.ForceReserve()
	if rp.ActiveModel() != "opus" {
		t.Errorf("with no reserves, should stay on opus: %q", rp.ActiveModel())
	}
}

func TestReserve_ForceReserve_AlreadyOnReserve(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.ForceReserve()
	if rp.ActiveModel() != "sonnet" {
		t.Fatalf("should be on sonnet: %q", rp.ActiveModel())
	}

	// Force again — already on reserve, should stay
	rp.ForceReserve()
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should stay on sonnet: %q", rp.ActiveModel())
	}
}

func TestReserve_AllRateLimitSignals(t *testing.T) {
	signals := []string{
		"Error: rate limit exceeded",
		"rate_limit_error occurred",
		"too many requests sent",
		"HTTP 429 response",
		"quota exceeded for model",
		"usage limit reached",
		"please try again later",
		"at capacity right now",
	}

	for _, sig := range signals {
		rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
		detected := rp.CheckOutput(sig)
		if !detected {
			t.Errorf("should detect rate limit in %q", sig)
		}
		if rp.ActiveModel() != "sonnet" {
			t.Errorf("should switch to sonnet for signal %q", sig)
		}
	}
}

func TestReserve_CaseInsensitive(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	detected := rp.CheckOutput("RATE LIMIT EXCEEDED")
	if !detected {
		t.Error("should detect rate limit case-insensitively")
	}
}

func TestReserve_NoFalsePositiveOnPartialMatches(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))

	// These should not trigger rate-limit detection.
	noMatch := []string{
		"corporate policy updated",
		"separate module update",
		"quota is mentioned without being exceeded",
		"capacity planning meeting",
		"usage is within limits",
		"try again? later we will see",
		"429th item processed",
	}
	for _, s := range noMatch {
		if rp.CheckOutput(s) {
			t.Errorf("should not detect rate limit for %q", s)
		}
	}
	if rp.ActiveModel() != "opus" {
		t.Errorf("should stay on opus, got %q", rp.ActiveModel())
	}
}

func TestReserve_ConcurrentAccess(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			rp.CheckOutput("normal output")
		}()
		go func() {
			defer wg.Done()
			_ = rp.ActiveModel()
		}()
		go func() {
			defer wg.Done()
			_ = rp.Status()
		}()
	}
	wg.Wait()

	// Should still be on primary since no rate limit signals
	if rp.ActiveModel() != "opus" {
		t.Error("should stay on opus with normal output")
	}
}
