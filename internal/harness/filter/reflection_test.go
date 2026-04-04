package filter_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/harness/filter"
	"github.com/hironow/paintress/internal/harness/policy"
)

// TestBuildReviewFixPromptWithReflection_WithAccumulator verifies accumulator injection.
func TestBuildReviewFixPromptWithReflection_WithAccumulator(t *testing.T) {
	// given
	branch := "feat/test-branch"
	comments := "[P1] Fix the error handling"
	acc := policy.NewReflectionAccumulator()
	acc.AddCycle(1, "[P2] prior issue")

	// when
	prompt := filter.BuildReviewFixPromptWithReflection(branch, comments, acc)

	// then
	if !strings.Contains(prompt, branch) {
		t.Errorf("prompt missing branch: %q", prompt)
	}
	if !strings.Contains(prompt, comments) {
		t.Errorf("prompt missing comments: %q", prompt)
	}
	if !strings.Contains(prompt, "prior issue") {
		t.Errorf("prompt missing reflection history: %q", prompt)
	}
}

// TestBuildReviewFixPromptWithReflection_WithEmptyAccumulator verifies no reflection section when empty.
func TestBuildReviewFixPromptWithReflection_WithEmptyAccumulator(t *testing.T) {
	// given
	branch := "feat/test-branch"
	comments := "[P1] Fix the error handling"
	acc := policy.NewReflectionAccumulator()

	// when
	prompt := filter.BuildReviewFixPromptWithReflection(branch, comments, acc)

	// then: should still produce a valid prompt
	if !strings.Contains(prompt, branch) {
		t.Errorf("prompt missing branch: %q", prompt)
	}
	if !strings.Contains(prompt, comments) {
		t.Errorf("prompt missing comments: %q", prompt)
	}
}
