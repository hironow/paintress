package domain_test

import (
	"encoding/json"
	"testing"
	"testing/quick"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// Property-based tests for GradientGauge invariants.
// These use testing/quick (stdlib, no external dependencies) to verify
// properties hold for arbitrary operation sequences.

// TestGradient_Property_LevelBounds verifies that the gauge level never
// goes below 0 or above max, regardless of the operation sequence.
func TestGradient_Property_LevelBounds(t *testing.T) {
	f := func(ops []byte, maxVal uint8) bool {
		max := int(maxVal%20) + 1 // max in [1, 20]
		g := domain.NewGradientGauge(max)

		for _, op := range ops {
			switch op % 3 {
			case 0:
				g.Charge()
			case 1:
				g.Discharge()
			case 2:
				g.Decay()
			}
			level := g.Level()
			if level < 0 || level > max {
				return false
			}
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("LevelBounds property violated: %v", err)
	}
}

// TestGradient_Property_IsGradientAttackConsistency verifies that
// IsGradientAttack() == true iff Level() >= max.
func TestGradient_Property_IsGradientAttackConsistency(t *testing.T) {
	f := func(ops []byte, maxVal uint8) bool {
		max := int(maxVal%20) + 1
		g := domain.NewGradientGauge(max)

		for _, op := range ops {
			switch op % 3 {
			case 0:
				g.Charge()
			case 1:
				g.Discharge()
			case 2:
				g.Decay()
			}
		}
		return g.IsGradientAttack() == (g.Level() >= max)
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 1000}); err != nil {
		t.Errorf("IsGradientAttack consistency violated: %v", err)
	}
}

// TestGradient_Property_PriorityHintMonotonicity verifies that as level
// increases, PriorityHint never "downgrades" (empty -> normal -> high -> attack).
func TestGradient_Property_PriorityHintMonotonicity(t *testing.T) {
	hintRank := func(hint string) int {
		switch {
		case containsStr(hint, "GRADIENT ATTACK"):
			return 3
		case containsStr(hint, "Gauge high"):
			return 2
		case containsStr(hint, "Normal"):
			return 1
		default: // "Gauge empty"
			return 0
		}
	}

	f := func(maxVal uint8) bool {
		max := int(maxVal%20) + 1
		g := domain.NewGradientGauge(max)

		prevRank := hintRank(g.PriorityHint())
		for i := 0; i < max; i++ {
			g.Charge()
			rank := hintRank(g.PriorityHint())
			if rank < prevRank {
				return false // downgrade detected
			}
			prevRank = rank
		}
		return true
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 500}); err != nil {
		t.Errorf("PriorityHint monotonicity violated: %v", err)
	}
}

// TestGradientGauge_ExpeditionAggregate_Linkage verifies that
// RecordGradientChange produces events with Level matching the gauge state.
func TestGradientGauge_ExpeditionAggregate_Linkage(t *testing.T) {
	// given
	gauge := domain.NewGradientGauge(5)
	agg := domain.NewExpeditionAggregate()
	now := time.Now()

	// when: charge gauge and record
	gauge.Charge()
	gauge.Charge()
	gauge.Charge()
	ev, err := agg.RecordGradientChange(gauge.Level(), "charge", now)

	// then
	if err != nil {
		t.Fatalf("RecordGradientChange: %v", err)
	}
	if ev.Type != domain.EventGradientChanged {
		t.Errorf("event type: got %s, want %s", ev.Type, domain.EventGradientChanged)
	}

	// Unmarshal and verify level matches gauge
	var data domain.GradientChangedData
	if unmarshalErr := json.Unmarshal(ev.Data, &data); unmarshalErr != nil {
		t.Fatalf("unmarshal GradientChangedData: %v", unmarshalErr)
	}
	if data.Level != 3 {
		t.Errorf("GradientChangedData.Level: got %d, want 3", data.Level)
	}
	if data.Operator != "charge" {
		t.Errorf("GradientChangedData.Operator: got %q, want %q", data.Operator, "charge")
	}
}

// TestGradientGauge_DischargeEvent verifies discharge produces level=0 event.
func TestGradientGauge_DischargeEvent(t *testing.T) {
	gauge := domain.NewGradientGauge(5)
	agg := domain.NewExpeditionAggregate()
	now := time.Now()

	gauge.Charge()
	gauge.Charge()
	gauge.Discharge()
	ev, err := agg.RecordGradientChange(gauge.Level(), "discharge", now)
	if err != nil {
		t.Fatalf("RecordGradientChange: %v", err)
	}

	var data domain.GradientChangedData
	json.Unmarshal(ev.Data, &data)
	if data.Level != 0 {
		t.Errorf("after discharge, Level: got %d, want 0", data.Level)
	}
}
