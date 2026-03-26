package domain_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

// TestReflectionAccumulator_EmptyAccumulator verifies FormatForPrompt on empty accumulator.
func TestReflectionAccumulator_EmptyAccumulator(t *testing.T) {
	// given
	acc := domain.NewReflectionAccumulator()

	// when
	prompt := acc.FormatForPrompt()

	// then: empty accumulator produces empty or minimal output
	if strings.TrimSpace(prompt) != "" {
		t.Errorf("FormatForPrompt on empty accumulator = %q, want empty string", prompt)
	}
}

// TestReflectionAccumulator_AddCycleAndFormat verifies cycle data is formatted.
func TestReflectionAccumulator_AddCycleAndFormat(t *testing.T) {
	// given
	acc := domain.NewReflectionAccumulator()
	acc.AddCycle(1, "[P2] Missing error handling\n[P1] Nil pointer risk")

	// when
	prompt := acc.FormatForPrompt()

	// then
	if prompt == "" {
		t.Fatal("FormatForPrompt should not be empty after adding a cycle")
	}
	if !strings.Contains(prompt, "cycle") && !strings.Contains(prompt, "Cycle") {
		t.Errorf("FormatForPrompt should mention cycle, got: %q", prompt)
	}
	if !strings.Contains(prompt, "Missing error handling") {
		t.Errorf("FormatForPrompt should contain review comments, got: %q", prompt)
	}
}

// TestReflectionAccumulator_MultipleCycles verifies all cycles are accumulated.
func TestReflectionAccumulator_MultipleCycles(t *testing.T) {
	// given
	acc := domain.NewReflectionAccumulator()
	acc.AddCycle(1, "[P2] First issue")
	acc.AddCycle(2, "[P1] Second issue")

	// when
	prompt := acc.FormatForPrompt()

	// then
	if !strings.Contains(prompt, "First issue") {
		t.Errorf("FormatForPrompt missing cycle 1 content: %q", prompt)
	}
	if !strings.Contains(prompt, "Second issue") {
		t.Errorf("FormatForPrompt missing cycle 2 content: %q", prompt)
	}
}

// TestReflectionAccumulator_IsStagnantAfterNoProgress verifies stagnation detection.
func TestReflectionAccumulator_IsStagnantAfterNoProgress(t *testing.T) {
	// given: same comment count across two cycles
	acc := domain.NewReflectionAccumulator()
	acc.AddCycle(1, "[P2] issue A\n[P2] issue B") // 2 tags
	acc.AddCycle(2, "[P2] still A\n[P2] still B") // 2 tags (no change)

	// when
	stagnant := acc.IsStagnant()

	// then
	if !stagnant {
		t.Error("IsStagnant = false, want true (no progress across cycles)")
	}
}

// TestReflectionAccumulator_IsStagnantAfterImprovement verifies no stagnation when improving.
func TestReflectionAccumulator_IsStagnantAfterImprovement(t *testing.T) {
	// given: tag count decreased
	acc := domain.NewReflectionAccumulator()
	acc.AddCycle(1, "[P2] issue A\n[P2] issue B\n[P1] issue C") // 3 tags
	acc.AddCycle(2, "[P2] issue A")                             // 1 tag (improvement)

	// when
	stagnant := acc.IsStagnant()

	// then
	if stagnant {
		t.Error("IsStagnant = true, want false (count decreased = improvement)")
	}
}

// TestReflectionAccumulator_IsStagnantWithOneCycle verifies no stagnation after first cycle.
func TestReflectionAccumulator_IsStagnantWithOneCycle(t *testing.T) {
	// given: only one cycle recorded
	acc := domain.NewReflectionAccumulator()
	acc.AddCycle(1, "[P2] issue A")

	// when
	stagnant := acc.IsStagnant()

	// then: cannot determine stagnation from one cycle
	if stagnant {
		t.Error("IsStagnant = true, want false (only one cycle — no baseline)")
	}
}

// TestBuildReviewFixPrompt_WithAccumulator verifies accumulator injection.
func TestBuildReviewFixPrompt_WithAccumulator(t *testing.T) {
	// given
	branch := "feat/test-branch"
	comments := "[P1] Fix the error handling"
	acc := domain.NewReflectionAccumulator()
	acc.AddCycle(1, "[P2] prior issue")

	// when
	prompt := domain.BuildReviewFixPromptWithReflection(branch, comments, acc)

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

// TestBuildReviewFixPrompt_WithEmptyAccumulator verifies no reflection section when empty.
func TestBuildReviewFixPrompt_WithEmptyAccumulator(t *testing.T) {
	// given
	branch := "feat/test-branch"
	comments := "[P1] Fix the error handling"
	acc := domain.NewReflectionAccumulator()

	// when
	prompt := domain.BuildReviewFixPromptWithReflection(branch, comments, acc)

	// then: should still produce a valid prompt
	if !strings.Contains(prompt, branch) {
		t.Errorf("prompt missing branch: %q", prompt)
	}
	if !strings.Contains(prompt, comments) {
		t.Errorf("prompt missing comments: %q", prompt)
	}
}
