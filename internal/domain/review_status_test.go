package domain_test

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestReviewGateSection_Passed(t *testing.T) {
	// given
	status := domain.ReviewGateStatus{
		Passed:    true,
		Cycle:     1,
		MaxCycles: 3,
	}

	// when
	section := status.FormatSection()

	// then
	if section == "" {
		t.Fatal("expected non-empty section")
	}
	if !containsAll(section, "## Review Gate", "PASSED", "1/3") {
		t.Errorf("unexpected section:\n%s", section)
	}
}

func TestReviewGateSection_NotResolved(t *testing.T) {
	// given
	status := domain.ReviewGateStatus{
		Passed:       false,
		Cycle:        3,
		MaxCycles:    3,
		LastComments: "some review comments here",
	}

	// when
	section := status.FormatSection()

	// then
	if !containsAll(section, "## Review Gate", "NOT RESOLVED", "3/3") {
		t.Errorf("unexpected section:\n%s", section)
	}
}

func TestReviewGateSection_Skipped(t *testing.T) {
	// given
	status := domain.ReviewGateStatus{
		Skipped: true,
	}

	// when
	section := status.FormatSection()

	// then
	if !containsAll(section, "## Review Gate", "SKIPPED") {
		t.Errorf("unexpected section:\n%s", section)
	}
}

func TestAppendReviewGateSection_NewSection(t *testing.T) {
	// given
	body := "## Summary\n\nSome PR description\n"
	section := "## Review Gate\n\n- Status: **PASSED**\n"

	// when
	result := domain.AppendReviewGateSection(body, section)

	// then
	if !containsAll(result, "## Summary", "## Review Gate", "PASSED") {
		t.Errorf("unexpected result:\n%s", result)
	}
}

func TestAppendReviewGateSection_ReplaceExisting(t *testing.T) {
	// given
	body := "## Summary\n\nSome PR description\n\n## Review Gate\n\n- Status: **OLD**\n"
	section := "## Review Gate\n\n- Status: **PASSED**\n"

	// when
	result := domain.AppendReviewGateSection(body, section)

	// then
	if !containsAll(result, "## Summary", "PASSED") {
		t.Errorf("unexpected result:\n%s", result)
	}
	if containsAll(result, "OLD") {
		t.Errorf("old section not replaced:\n%s", result)
	}
}

func TestAppendReviewGateSection_PreservesBlankLineBetweenSections(t *testing.T) {
	// given: body with Summary, Review Gate, and Changelog sections.
	// The section replacement has no trailing newline, exposing the TrimLeft bug
	// where the blank-line separator between ## headers is lost.
	body := "## Summary\n\nSome PR description\n\n## Review Gate\n\n- Status: **OLD**\n\n## Changelog\n\nEntries here\n"
	section := "## Review Gate\n\n- Status: **PASSED**"

	// when
	result := domain.AppendReviewGateSection(body, section)

	// then: blank line between Review Gate and Changelog must be preserved
	if !strings.Contains(result, "\n\n## Changelog") {
		t.Errorf("blank line before ## Changelog is missing:\n%s", result)
	}
	if !strings.Contains(result, "Entries here") {
		t.Errorf("Changelog content lost:\n%s", result)
	}
	if strings.Contains(result, "OLD") {
		t.Errorf("old section not replaced:\n%s", result)
	}
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}

// --- MY-492: ReviewCycleHistory ---

func TestReviewCycleHistory_ImprovementRate_TypicalImprovement(t *testing.T) {
	// given: initial 10 comments, final 4 comments
	h := domain.ReviewCycleHistory{
		InitialCommentCount: 10,
		FinalCommentCount:   4,
	}

	// when
	rate := h.ImprovementRate()

	// then: (10-4)/10 = 0.6
	if rate < 0.599 || rate > 0.601 {
		t.Errorf("ImprovementRate() = %f, want ~0.6", rate)
	}
}

func TestReviewCycleHistory_ImprovementRate_ZeroInitial(t *testing.T) {
	// given: initial 0 (no comments at start)
	h := domain.ReviewCycleHistory{
		InitialCommentCount: 0,
		FinalCommentCount:   0,
	}

	// when
	rate := h.ImprovementRate()

	// then: zero initial should return 0.0 (not divide by zero)
	if rate != 0.0 {
		t.Errorf("ImprovementRate() with zero initial = %f, want 0.0", rate)
	}
}

func TestReviewCycleHistory_ImprovementRate_NoImprovement(t *testing.T) {
	// given: same count start and end
	h := domain.ReviewCycleHistory{
		InitialCommentCount: 5,
		FinalCommentCount:   5,
	}

	// when
	rate := h.ImprovementRate()

	// then: 0.0 — no improvement
	if rate != 0.0 {
		t.Errorf("ImprovementRate() no improvement = %f, want 0.0", rate)
	}
}

func TestReviewCycleHistory_IsStalled_WithinWindow(t *testing.T) {
	// given: improvement rate is low (5 → 5), stall window 3 cycles
	h := domain.ReviewCycleHistory{
		InitialCommentCount: 5,
		FinalCommentCount:   5,
		CycleCount:          3,
	}

	// when
	stalled := h.IsStalled(3)

	// then: no improvement over 3 cycles is stalled
	if !stalled {
		t.Error("IsStalled(3) = false, want true (no improvement over stallWindow cycles)")
	}
}

func TestReviewCycleHistory_IsStalled_BelowWindow(t *testing.T) {
	// given: only 1 cycle so far
	h := domain.ReviewCycleHistory{
		InitialCommentCount: 5,
		FinalCommentCount:   5,
		CycleCount:          1,
	}

	// when
	stalled := h.IsStalled(3)

	// then: not enough cycles to declare stall
	if stalled {
		t.Error("IsStalled(3) = true, want false (below stallWindow)")
	}
}

func TestReviewCycleHistory_IsStalled_WithImprovement(t *testing.T) {
	// given: good improvement, enough cycles
	h := domain.ReviewCycleHistory{
		InitialCommentCount: 10,
		FinalCommentCount:   2,
		CycleCount:          4,
	}

	// when
	stalled := h.IsStalled(3)

	// then: improving — not stalled
	if stalled {
		t.Error("IsStalled(3) = true, want false (has improvement)")
	}
}

func TestReviewCycleHistory_FormatStallWarning_ContainsKeyInfo(t *testing.T) {
	// given
	h := domain.ReviewCycleHistory{
		InitialCommentCount: 8,
		FinalCommentCount:   7,
		CycleCount:          4,
	}

	// when
	warning := h.FormatStallWarning()

	// then
	if warning == "" {
		t.Fatal("FormatStallWarning() should not be empty")
	}
	if !strings.Contains(warning, "stall") && !strings.Contains(warning, "Stall") && !strings.Contains(warning, "STALL") {
		t.Errorf("FormatStallWarning() should mention stall: %q", warning)
	}
}
