package domain_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

// TestStrategyForCycle_Cycle1ReturnsDirect verifies first cycle uses Direct strategy.
func TestStrategyForCycle_Cycle1ReturnsDirect(t *testing.T) {
	// when
	strategy := domain.StrategyForCycle(1)

	// then
	if strategy != domain.StrategyDirect {
		t.Errorf("StrategyForCycle(1) = %v, want StrategyDirect", strategy)
	}
}

// TestStrategyForCycle_Cycle2ReturnsDecompose verifies second cycle uses Decompose strategy.
func TestStrategyForCycle_Cycle2ReturnsDecompose(t *testing.T) {
	// when
	strategy := domain.StrategyForCycle(2)

	// then
	if strategy != domain.StrategyDecompose {
		t.Errorf("StrategyForCycle(2) = %v, want StrategyDecompose", strategy)
	}
}

// TestStrategyForCycle_Cycle3ReturnsRewrite verifies third cycle uses Rewrite strategy.
func TestStrategyForCycle_Cycle3ReturnsRewrite(t *testing.T) {
	// when
	strategy := domain.StrategyForCycle(3)

	// then
	if strategy != domain.StrategyRewrite {
		t.Errorf("StrategyForCycle(3) = %v, want StrategyRewrite", strategy)
	}
}

// TestStrategyForCycle_RotatesAfterCycle3 verifies cycle 4 wraps back to Direct.
func TestStrategyForCycle_RotatesAfterCycle3(t *testing.T) {
	// when: cycle 4 should rotate back to Direct
	strategy := domain.StrategyForCycle(4)

	// then
	if strategy != domain.StrategyDirect {
		t.Errorf("StrategyForCycle(4) = %v, want StrategyDirect (rotation)", strategy)
	}
}

// TestStrategyForCycle_RotationPattern verifies the full rotation pattern.
func TestStrategyForCycle_RotationPattern(t *testing.T) {
	// given: expected rotation
	expected := []domain.FixStrategy{
		domain.StrategyDirect,   // 1
		domain.StrategyDecompose, // 2
		domain.StrategyRewrite,   // 3
		domain.StrategyDirect,   // 4
		domain.StrategyDecompose, // 5
		domain.StrategyRewrite,   // 6
	}

	for i, want := range expected {
		cycle := i + 1
		got := domain.StrategyForCycle(cycle)
		if got != want {
			t.Errorf("StrategyForCycle(%d) = %v, want %v", cycle, got, want)
		}
	}
}

// TestBuildReviewFixPromptWithStrategy_DirectHint verifies Direct strategy produces no extra hint.
func TestBuildReviewFixPromptWithStrategy_DirectHint(t *testing.T) {
	// given
	branch := "feat/test"
	comments := "[P1] Fix error handling"

	// when
	prompt := domain.BuildReviewFixPromptWithStrategy(branch, comments, domain.StrategyDirect)

	// then: direct strategy should still contain the core content
	if !strings.Contains(prompt, branch) {
		t.Errorf("prompt missing branch: %q", prompt)
	}
	if !strings.Contains(prompt, comments) {
		t.Errorf("prompt missing comments: %q", prompt)
	}
}

// TestBuildReviewFixPromptWithStrategy_DecomposeHint verifies Decompose strategy injects hint.
func TestBuildReviewFixPromptWithStrategy_DecomposeHint(t *testing.T) {
	// given
	branch := "feat/test"
	comments := "[P1] Fix error handling"

	// when
	prompt := domain.BuildReviewFixPromptWithStrategy(branch, comments, domain.StrategyDecompose)

	// then: decompose strategy should inject decomposition hint
	if !strings.Contains(prompt, branch) {
		t.Errorf("prompt missing branch: %q", prompt)
	}
	lowerPrompt := strings.ToLower(prompt)
	if !strings.Contains(lowerPrompt, "decompose") && !strings.Contains(lowerPrompt, "break") && !strings.Contains(lowerPrompt, "step") {
		t.Errorf("Decompose strategy should inject decomposition hint, got: %q", prompt)
	}
}

// TestBuildReviewFixPromptWithStrategy_RewriteHint verifies Rewrite strategy injects hint.
func TestBuildReviewFixPromptWithStrategy_RewriteHint(t *testing.T) {
	// given
	branch := "feat/test"
	comments := "[P1] Fix error handling"

	// when
	prompt := domain.BuildReviewFixPromptWithStrategy(branch, comments, domain.StrategyRewrite)

	// then: rewrite strategy should inject rewrite hint
	if !strings.Contains(prompt, branch) {
		t.Errorf("prompt missing branch: %q", prompt)
	}
	lowerPrompt := strings.ToLower(prompt)
	if !strings.Contains(lowerPrompt, "rewrite") && !strings.Contains(lowerPrompt, "fresh") && !strings.Contains(lowerPrompt, "from scratch") {
		t.Errorf("Rewrite strategy should inject rewrite hint, got: %q", prompt)
	}
}
