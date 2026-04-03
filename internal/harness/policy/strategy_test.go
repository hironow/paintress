package policy_test

import (
	"testing"

	"github.com/hironow/paintress/internal/harness/policy"
)

// TestStrategyForCycle_Cycle1ReturnsDirect verifies first cycle uses Direct strategy.
func TestStrategyForCycle_Cycle1ReturnsDirect(t *testing.T) {
	// when
	strategy := policy.StrategyForCycle(1)

	// then
	if strategy != policy.StrategyDirect {
		t.Errorf("StrategyForCycle(1) = %v, want StrategyDirect", strategy)
	}
}

// TestStrategyForCycle_Cycle2ReturnsDecompose verifies second cycle uses Decompose strategy.
func TestStrategyForCycle_Cycle2ReturnsDecompose(t *testing.T) {
	// when
	strategy := policy.StrategyForCycle(2)

	// then
	if strategy != policy.StrategyDecompose {
		t.Errorf("StrategyForCycle(2) = %v, want StrategyDecompose", strategy)
	}
}

// TestStrategyForCycle_Cycle3ReturnsRewrite verifies third cycle uses Rewrite strategy.
func TestStrategyForCycle_Cycle3ReturnsRewrite(t *testing.T) {
	// when
	strategy := policy.StrategyForCycle(3)

	// then
	if strategy != policy.StrategyRewrite {
		t.Errorf("StrategyForCycle(3) = %v, want StrategyRewrite", strategy)
	}
}

// TestStrategyForCycle_RotatesAfterCycle3 verifies cycle 4 wraps back to Direct.
func TestStrategyForCycle_RotatesAfterCycle3(t *testing.T) {
	// when: cycle 4 should rotate back to Direct
	strategy := policy.StrategyForCycle(4)

	// then
	if strategy != policy.StrategyDirect {
		t.Errorf("StrategyForCycle(4) = %v, want StrategyDirect (rotation)", strategy)
	}
}

// TestStrategyForCycle_RotationPattern verifies the full rotation pattern.
func TestStrategyForCycle_RotationPattern(t *testing.T) {
	// given: expected rotation
	expected := []policy.FixStrategy{
		policy.StrategyDirect,    // 1
		policy.StrategyDecompose, // 2
		policy.StrategyRewrite,   // 3
		policy.StrategyDirect,    // 4
		policy.StrategyDecompose, // 5
		policy.StrategyRewrite,   // 6
	}

	for i, want := range expected {
		cycle := i + 1
		got := policy.StrategyForCycle(cycle)
		if got != want {
			t.Errorf("StrategyForCycle(%d) = %v, want %v", cycle, got, want)
		}
	}
}

