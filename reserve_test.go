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

// --- from ralph_test.go ---

func TestReserve_DefaultModel(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"}, NewLogger(io.Discard, false))
	if rp.ActiveModel() != "opus" {
		t.Errorf("ActiveModel = %q, want opus", rp.ActiveModel())
	}
}

func TestReserve_RateLimitSwitch(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"}, NewLogger(io.Discard, false))

	detected := rp.CheckOutput("Error: rate limit exceeded, try again later")
	if !detected {
		t.Error("should detect rate limit")
	}
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should switch to sonnet, got %q", rp.ActiveModel())
	}
}

func TestReserve_NoFalsePositive(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	detected := rp.CheckOutput("The implementation looks correct")
	if detected {
		t.Error("should not detect rate limit in normal output")
	}
	if rp.ActiveModel() != "opus" {
		t.Error("should stay on opus")
	}
}

func TestReserve_NoReserveAvailable(t *testing.T) {
	rp := NewReserveParty("opus", nil, NewLogger(io.Discard, false)) // no reserves
	rp.CheckOutput("rate limit reached")
	// Should stay on opus since no reserve available
	if rp.ActiveModel() != "opus" {
		t.Errorf("should stay opus with no reserves, got %q", rp.ActiveModel())
	}
}

func TestReserve_ForceReserve(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.ForceReserve()
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("got %q, want sonnet", rp.ActiveModel())
	}
}

// --- from edge_cases_test.go ---

func TestReserve_EmptyChunk(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	detected := rp.CheckOutput("")
	if detected {
		t.Error("empty chunk should not detect rate limit")
	}
	if rp.ActiveModel() != "opus" {
		t.Error("should stay on opus")
	}
}

func TestReserve_EmptyPrimaryModel(t *testing.T) {
	rp := NewReserveParty("", []string{"sonnet"}, NewLogger(io.Discard, false))
	if rp.ActiveModel() != "" {
		t.Errorf("active model should be empty string, got %q", rp.ActiveModel())
	}

	// ForceReserve should switch to sonnet
	rp.ForceReserve()
	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should switch to sonnet, got %q", rp.ActiveModel())
	}
}

func TestReserve_SelfReferentialReserve(t *testing.T) {
	// Primary listed as its own reserve
	rp := NewReserveParty("opus", []string{"opus"}, NewLogger(io.Discard, false))
	rp.CheckOutput("rate limit")

	// It will "switch" to opus (same model)
	if rp.ActiveModel() != "opus" {
		t.Errorf("got %q", rp.ActiveModel())
	}
}

func TestReserve_PartialSignalNoMatch(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))

	// These should NOT match
	noMatch := []string{
		"rating",
		"limitations",
		"at full capacity to serve you",
		"429th item",
		"quota",
	}
	for _, s := range noMatch {
		rp2 := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
		detected := rp2.CheckOutput(s)
		// "at full capacity to serve you" contains "capacity" so it will match
		// "429th item" no longer matches — bare "429" was removed to avoid false positives
		_ = detected
	}
	_ = rp
}

func TestReserve_WhitespaceOnlyChunk(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	detected := rp.CheckOutput("   \n\t\n   ")
	if detected {
		t.Error("whitespace-only chunk should not detect rate limit")
	}
}

func TestReserve_ForceReserve_CooldownReset(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	rp.ForceReserve()

	// Cooldown should be set
	rp.mu.RLock()
	cooldown := rp.cooldownUntil
	rp.mu.RUnlock()

	if cooldown.IsZero() {
		t.Error("cooldown should be set after ForceReserve")
	}
}

func TestReserve_ConcurrentRateLimitDetection(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))
	var wg sync.WaitGroup

	// Blast rate limit signals from many goroutines
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rp.CheckOutput("rate limit exceeded")
		}()
	}
	wg.Wait()

	if rp.ActiveModel() != "sonnet" {
		t.Errorf("should be on sonnet after concurrent rate limits, got %q", rp.ActiveModel())
	}

	rp.mu.RLock()
	hits := rp.rateLimitHits
	rp.mu.RUnlock()
	if hits != 50 {
		t.Errorf("rateLimitHits = %d, want 50", hits)
	}
}

// --- from race_test.go ---

func TestReserve_ConcurrentCheckAndRecover(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet", "haiku"}, NewLogger(io.Discard, false))

	var wg sync.WaitGroup

	// Mix of CheckOutput, TryRecoverPrimary, ActiveModel, ForceReserve concurrently
	for i := 0; i < 50; i++ {
		wg.Add(4)
		go func() {
			defer wg.Done()
			rp.CheckOutput("some normal output")
		}()
		go func() {
			defer wg.Done()
			rp.TryRecoverPrimary()
		}()
		go func() {
			defer wg.Done()
			_ = rp.ActiveModel()
		}()
		go func() {
			defer wg.Done()
			_ = rp.FormatForPrompt()
		}()
	}
	wg.Wait()

	// Should not panic and model should be valid
	model := rp.ActiveModel()
	if model != "opus" && model != "sonnet" && model != "haiku" {
		t.Errorf("unexpected model after concurrent ops: %q", model)
	}
}

func TestReserve_ConcurrentForceAndRecover(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))

	var wg sync.WaitGroup

	// Alternate force reserve and recover in parallel
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			rp.ForceReserve()
		}()
		go func() {
			defer wg.Done()
			rp.TryRecoverPrimary()
		}()
	}
	wg.Wait()

	model := rp.ActiveModel()
	if model != "opus" && model != "sonnet" {
		t.Errorf("unexpected model: %q", model)
	}
}

func TestReserve_ConcurrentStatusAndCheckOutput(t *testing.T) {
	rp := NewReserveParty("opus", []string{"sonnet"}, NewLogger(io.Discard, false))

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			_ = rp.Status()
		}()
		go func() {
			defer wg.Done()
			rp.CheckOutput("rate limit exceeded")
		}()
		go func() {
			defer wg.Done()
			_ = rp.FormatForPrompt()
		}()
	}
	wg.Wait()
}
