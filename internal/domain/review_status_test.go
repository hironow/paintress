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

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
