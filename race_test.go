package paintress

import (
	"bytes"
	"io"
	"sync"
	"testing"
)

// === Reserve Party: Extended Concurrent Scenarios ===

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

// === Gradient Gauge: Extended Concurrent Scenarios ===

func TestGradient_ConcurrentFormatForPrompt(t *testing.T) {
	g := NewGradientGauge(5)

	var wg sync.WaitGroup

	// FormatForPrompt calls priorityHint internally (deadlock regression test)
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			g.Charge()
		}()
		go func() {
			defer wg.Done()
			_ = g.FormatForPrompt()
		}()
		go func() {
			defer wg.Done()
			_ = g.PriorityHint()
		}()
	}
	wg.Wait()

	level := g.Level()
	if level < 0 || level > 5 {
		t.Errorf("level out of range: %d", level)
	}
}

func TestGradient_ConcurrentFormatLog(t *testing.T) {
	g := NewGradientGauge(10)

	var wg sync.WaitGroup

	for i := 0; i < 30; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			g.Charge()
		}()
		go func() {
			defer wg.Done()
			g.Decay()
		}()
		go func() {
			defer wg.Done()
			_ = g.FormatLog()
		}()
	}
	wg.Wait()

	log := g.FormatLog()
	if log == "" {
		t.Error("log should not be empty after operations")
	}
}

// === Logger: Concurrent SetExtraWriter/Write ===

func TestLogger_ConcurrentSetExtraWriterAndWrite(t *testing.T) {
	logger := NewLogger(io.Discard, false)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			var buf bytes.Buffer
			logger.SetExtraWriter(&buf)
		}()
		go func(n int) {
			defer wg.Done()
			logger.Info("race test info %d", n)
			logger.Warn("race test warn %d", n)
		}(i)
		go func() {
			defer wg.Done()
			logger.SetExtraWriter(nil)
		}()
	}
	wg.Wait()

	// Clean up
	logger.SetExtraWriter(nil)
}

// === Lang: Concurrent Msg() reads with Lang writes ===

func TestLang_ConcurrentMsgReads(t *testing.T) {
	orig := Lang
	defer func() { Lang = orig }()

	var wg sync.WaitGroup

	// Concurrent reads of Msg() — all reading the same global Lang
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg := Msg("grad_attack")
			if msg == "" {
				t.Error("Msg should never return empty")
			}
		}()
	}
	wg.Wait()
}

