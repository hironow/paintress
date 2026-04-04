package policy_test

import (
	"testing"

	"github.com/hironow/paintress/internal/harness/policy"
)

// TestCountPriorityTags_Empty verifies zero tags returns zero count.
func TestCountPriorityTags_Empty(t *testing.T) {
	// given
	output := "No issues found"

	// when
	count := policy.CountPriorityTags(output)

	// then
	if count != 0 {
		t.Errorf("CountPriorityTags = %d, want 0", count)
	}
}

// TestCountPriorityTags_SingleTag verifies a single P0 tag counts as 1.
func TestCountPriorityTags_SingleTag(t *testing.T) {
	// given
	output := "[P0] Critical bug: missing nil check"

	// when
	count := policy.CountPriorityTags(output)

	// then
	if count != 1 {
		t.Errorf("CountPriorityTags = %d, want 1", count)
	}
}

// TestCountPriorityTags_MultipleTagsAllPriorities verifies counts across P0-P4.
func TestCountPriorityTags_MultipleTagsAllPriorities(t *testing.T) {
	// given: one of each priority level
	output := "[P0] bug\n[P1] warning\n[P2] style\n[P3] suggestion\n[P4] nitpick"

	// when
	count := policy.CountPriorityTags(output)

	// then
	if count != 5 {
		t.Errorf("CountPriorityTags = %d, want 5", count)
	}
}

// TestCountPriorityTags_DuplicateTags verifies each occurrence is counted.
func TestCountPriorityTags_DuplicateTags(t *testing.T) {
	// given: same tag appears three times
	output := "[P2] issue A\n[P2] issue B\n[P2] issue C"

	// when
	count := policy.CountPriorityTags(output)

	// then
	if count != 3 {
		t.Errorf("CountPriorityTags = %d, want 3", count)
	}
}

// TestIsStagnant_NoPreviousCount verifies no stagnation when previousCount is zero.
func TestIsStagnant_NoPreviousCount(t *testing.T) {
	// given: first cycle, no previous count
	currentCount := 3
	previousCount := 0

	// when
	stagnant := policy.IsStagnant(currentCount, previousCount)

	// then: cannot be stagnant with no baseline
	if stagnant {
		t.Error("IsStagnant = true, want false (no previous count)")
	}
}

// TestIsStagnant_CountDecreased verifies improvement is not stagnation.
func TestIsStagnant_CountDecreased(t *testing.T) {
	// given: comment count decreased (improvement)
	currentCount := 2
	previousCount := 5

	// when
	stagnant := policy.IsStagnant(currentCount, previousCount)

	// then
	if stagnant {
		t.Error("IsStagnant = true, want false (count decreased = improvement)")
	}
}

// TestIsStagnant_CountUnchanged verifies same count signals stagnation.
func TestIsStagnant_CountUnchanged(t *testing.T) {
	// given: same count across two cycles
	currentCount := 4
	previousCount := 4

	// when
	stagnant := policy.IsStagnant(currentCount, previousCount)

	// then
	if !stagnant {
		t.Error("IsStagnant = false, want true (count unchanged)")
	}
}

// TestIsStagnant_CountIncreased verifies increasing count also signals stagnation.
func TestIsStagnant_CountIncreased(t *testing.T) {
	// given: count went up (fix made things worse or review re-found issues)
	currentCount := 6
	previousCount := 4

	// when
	stagnant := policy.IsStagnant(currentCount, previousCount)

	// then: not decreasing means stagnant
	if !stagnant {
		t.Error("IsStagnant = false, want true (count increased = no progress)")
	}
}

// TestIsStagnant_BothZero verifies all-clear is not stagnation.
func TestIsStagnant_BothZero(t *testing.T) {
	// given: no comments in either cycle (resolved)
	currentCount := 0
	previousCount := 0

	// when
	stagnant := policy.IsStagnant(currentCount, previousCount)

	// then: zero comments means passed, not stagnant
	if stagnant {
		t.Error("IsStagnant = true, want false (both zero = resolved)")
	}
}
