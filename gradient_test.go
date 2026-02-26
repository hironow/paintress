package paintress

import (
	"sync"
	"testing"
)

func TestGradient_NewStartsAtZero(t *testing.T) {
	g := NewGradientGauge(10)
	if g.Level() != 0 {
		t.Errorf("new gauge should start at 0, got %d", g.Level())
	}
	if g.IsGradientAttack() {
		t.Error("new gauge should not be gradient attack")
	}
}

func TestGradient_FormatForPrompt_AtZero(t *testing.T) {
	g := NewGradientGauge(3)
	s := g.FormatForPrompt()
	if !containsStr(s, "░░░") {
		t.Errorf("at 0, should show all empty bars: %q", s)
	}
	if !containsStr(s, "0/3") {
		t.Errorf("should show 0/3: %q", s)
	}
	if !containsStr(s, "Gauge empty") {
		t.Errorf("should suggest small issues: %q", s)
	}
}

func TestGradient_FormatForPrompt_AtMax(t *testing.T) {
	g := NewGradientGauge(3)
	g.Charge()
	g.Charge()
	g.Charge()
	s := g.FormatForPrompt()
	if !containsStr(s, "███") {
		t.Errorf("at max, should show all filled bars: %q", s)
	}
	if !containsStr(s, "3/3") {
		t.Errorf("should show 3/3: %q", s)
	}
	if !containsStr(s, "GRADIENT ATTACK") {
		t.Errorf("should say gradient attack: %q", s)
	}
}

func TestGradient_FormatForPrompt_Partial(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	s := g.FormatForPrompt()
	if !containsStr(s, "██░░░") {
		t.Errorf("at 2/5, should show 2 filled + 3 empty: %q", s)
	}
}

func TestGradient_FormatLog_Empty(t *testing.T) {
	g := NewGradientGauge(5)
	log := g.FormatLog()
	if !containsStr(log, "(empty)") {
		t.Errorf("empty gauge log should say empty: %q", log)
	}
}

func TestGradient_FormatLog_WithHistory(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Discharge()
	log := g.FormatLog()
	if !containsStr(log, "+1") {
		t.Errorf("log should contain charge: %q", log)
	}
	if !containsStr(log, "RESET") {
		t.Errorf("log should contain discharge: %q", log)
	}
}

func TestGradient_DecayLog(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Decay()
	log := g.FormatLog()
	if !containsStr(log, "-1") {
		t.Errorf("log should contain decay: %q", log)
	}
}

func TestGradient_ConcurrentAccess(t *testing.T) {
	g := NewGradientGauge(100)
	var wg sync.WaitGroup

	// 50 goroutines charge, 50 goroutines read level
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			g.Charge()
		}()
		go func() {
			defer wg.Done()
			_ = g.Level()
			_ = g.IsGradientAttack()
			_ = g.PriorityHint()
		}()
	}
	wg.Wait()

	level := g.Level()
	if level < 0 || level > 100 {
		t.Errorf("level out of range after concurrent access: %d", level)
	}
}

func TestGradient_MaxOfOne(t *testing.T) {
	g := NewGradientGauge(1)
	g.Charge()
	if !g.IsGradientAttack() {
		t.Error("max=1, single charge should be gradient attack")
	}
	if g.Level() != 1 {
		t.Errorf("Level = %d, want 1", g.Level())
	}
}

func TestGradient_ChargeDischargeCharge(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	g.Charge()    // level 3
	g.Discharge() // level 0
	g.Charge()    // level 1

	if g.Level() != 1 {
		t.Errorf("after charge-discharge-charge, Level = %d, want 1", g.Level())
	}
}

func TestGradient_PriorityHint_AllLevels(t *testing.T) {
	tests := []struct {
		charges int
		want    string
	}{
		{0, "Gauge empty"},
		{1, "Normal"},
		{2, "Normal"},
		{3, "Gauge high"},
		{4, "Gauge high"},
		{5, "GRADIENT ATTACK"},
	}

	for _, tt := range tests {
		g := NewGradientGauge(5)
		for i := 0; i < tt.charges; i++ {
			g.Charge()
		}
		hint := g.PriorityHint()
		if !containsStr(hint, tt.want) {
			t.Errorf("at level %d, want hint containing %q, got %q", tt.charges, tt.want, hint)
		}
	}
}

// --- from ralph_test.go ---

func TestGradient_Charge(t *testing.T) {
	g := NewGradientGauge(5)
	if g.Level() != 0 {
		t.Fatal("should start at 0")
	}

	g.Charge()
	g.Charge()
	g.Charge()
	if g.Level() != 3 {
		t.Errorf("Level = %d, want 3", g.Level())
	}
	if g.IsGradientAttack() {
		t.Error("should not be gradient attack at 3/5")
	}
}

func TestGradient_Full(t *testing.T) {
	g := NewGradientGauge(3)
	g.Charge()
	g.Charge()
	g.Charge()
	if !g.IsGradientAttack() {
		t.Error("should be gradient attack at 3/3")
	}
	// Should not exceed max
	g.Charge()
	if g.Level() != 3 {
		t.Errorf("should cap at max, got %d", g.Level())
	}
}

func TestGradient_Discharge(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	g.Charge()
	g.Discharge()
	if g.Level() != 0 {
		t.Errorf("should reset to 0, got %d", g.Level())
	}
}

func TestGradient_Decay(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	g.Decay()
	if g.Level() != 1 {
		t.Errorf("Level = %d, want 1", g.Level())
	}
	// Decay at 0 should stay 0
	g.Decay()
	g.Decay()
	if g.Level() != 0 {
		t.Errorf("should not go below 0, got %d", g.Level())
	}
}

func TestGradient_PriorityHint(t *testing.T) {
	g := NewGradientGauge(5)

	hint := g.PriorityHint()
	if !containsStr(hint, "Gauge empty") {
		t.Errorf("at 0, hint should suggest small issues: %s", hint)
	}

	g.Charge()
	hint = g.PriorityHint()
	if !containsStr(hint, "Normal") {
		t.Errorf("at 1, hint should be normal: %s", hint)
	}

	g.Charge()
	g.Charge()
	g.Charge() // level 4
	hint = g.PriorityHint()
	if !containsStr(hint, "Gauge high") {
		t.Errorf("at 4, hint should mention high priority: %s", hint)
	}

	g.Charge() // level 5 = max
	hint = g.PriorityHint()
	if !containsStr(hint, "GRADIENT ATTACK") {
		t.Errorf("at max, hint should say gradient attack: %s", hint)
	}
}

// --- from edge_cases_test.go ---

func TestGradient_MaxZero(t *testing.T) {
	g := NewGradientGauge(0)

	// At max=0, gauge is already "full" — IsGradientAttack should be true (level >= max)
	if !g.IsGradientAttack() {
		t.Error("max=0: level(0) >= max(0) should be gradient attack")
	}

	// Charge should be no-op (level < max is false when both are 0)
	g.Charge()
	if g.Level() != 0 {
		t.Errorf("max=0: Charge should be no-op, got level %d", g.Level())
	}

	// Discharge should work
	g.Discharge()
	if g.Level() != 0 {
		t.Errorf("max=0: Discharge should keep at 0, got %d", g.Level())
	}

	// FormatForPrompt should not panic
	s := g.FormatForPrompt()
	if s == "" {
		t.Error("FormatForPrompt should not return empty")
	}
}

func TestGradient_DischargeAtZero(t *testing.T) {
	g := NewGradientGauge(5)
	// Already at 0, discharge should be idempotent
	g.Discharge()
	if g.Level() != 0 {
		t.Errorf("Discharge at 0 should stay 0, got %d", g.Level())
	}

	// Log should record the reset
	log := g.FormatLog()
	if !containsStr(log, "RESET 0") {
		t.Errorf("log should record 0->0 reset: %q", log)
	}
}

func TestGradient_DoubleDischarge(t *testing.T) {
	g := NewGradientGauge(5)
	g.Charge()
	g.Charge()
	g.Charge()
	g.Discharge()
	g.Discharge() // second discharge at 0

	if g.Level() != 0 {
		t.Errorf("double discharge should be at 0, got %d", g.Level())
	}
}

func TestGradient_ConcurrentMixedOperations(t *testing.T) {
	g := NewGradientGauge(100)
	var wg sync.WaitGroup

	// Mix of Charge, Discharge, Decay concurrently
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
			if i%10 == 0 {
				g.Discharge()
			}
			_ = g.FormatForPrompt()
			_ = g.FormatLog()
		}()
	}
	wg.Wait()

	level := g.Level()
	if level < 0 || level > 100 {
		t.Errorf("level out of range after mixed concurrent ops: %d", level)
	}
}

func TestGradient_LargeMax(t *testing.T) {
	g := NewGradientGauge(1000)

	for i := 0; i < 1000; i++ {
		g.Charge()
	}
	if g.Level() != 1000 {
		t.Errorf("Level = %d, want 1000", g.Level())
	}
	if !g.IsGradientAttack() {
		t.Error("should be gradient attack at max")
	}

	s := g.FormatForPrompt()
	if !containsStr(s, "1000/1000") {
		t.Errorf("should show 1000/1000: %q", s)
	}
}

// --- from race_test.go ---

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
