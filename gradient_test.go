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
