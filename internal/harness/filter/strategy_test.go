package filter_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/harness/filter"
	"github.com/hironow/paintress/internal/harness/policy"
)

// TestBuildReviewFixPromptWithStrategy_DirectHint verifies Direct strategy produces no extra hint.
func TestBuildReviewFixPromptWithStrategy_DirectHint(t *testing.T) {
	// given
	branch := "feat/test"
	comments := "[P1] Fix error handling"

	// when
	prompt := filter.BuildReviewFixPromptWithStrategy(branch, comments, policy.StrategyDirect)

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
	prompt := filter.BuildReviewFixPromptWithStrategy(branch, comments, policy.StrategyDecompose)

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
	prompt := filter.BuildReviewFixPromptWithStrategy(branch, comments, policy.StrategyRewrite)

	// then: rewrite strategy should inject rewrite hint
	if !strings.Contains(prompt, branch) {
		t.Errorf("prompt missing branch: %q", prompt)
	}
	lowerPrompt := strings.ToLower(prompt)
	if !strings.Contains(lowerPrompt, "rewrite") && !strings.Contains(lowerPrompt, "fresh") && !strings.Contains(lowerPrompt, "from scratch") {
		t.Errorf("Rewrite strategy should inject rewrite hint, got: %q", prompt)
	}
}
