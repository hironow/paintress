package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestClassifyGommage_Timeout(t *testing.T) {
	reasons := []string{"timeout after 120s", "timeout after 120s", "timeout after 120s"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassTimeout {
		t.Errorf("got %q, want %q", got, domain.GommageClassTimeout)
	}
}

func TestClassifyGommage_RateLimit(t *testing.T) {
	reasons := []string{"rate_limit: model overloaded", "rate_limit: 429", "rate_limit: quota"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassRateLimit {
		t.Errorf("got %q, want %q", got, domain.GommageClassRateLimit)
	}
}

func TestClassifyGommage_ParseError(t *testing.T) {
	reasons := []string{"parse_error: no markers", "parse_error: invalid json", "parse_error: truncated"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassParseError {
		t.Errorf("got %q, want %q", got, domain.GommageClassParseError)
	}
}

func TestClassifyGommage_ParseError_JournalReason(t *testing.T) {
	// Real journal reason text: "report markers not found"
	reasons := []string{"report markers not found", "report markers not found", "report markers not found"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassParseError {
		t.Errorf("journal reason 'markers not found' should classify as parse_error, got %q", got)
	}
}

func TestClassifyGommage_Blocker(t *testing.T) {
	reasons := []string{"blocker: PR stuck", "blocker: merge conflict", "blocker: CI failed"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassBlocker {
		t.Errorf("got %q, want %q", got, domain.GommageClassBlocker)
	}
}

func TestClassifyGommage_Systematic(t *testing.T) {
	reasons := []string{"unknown error A", "unknown error B", "unknown error C"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassSystematic {
		t.Errorf("got %q, want %q", got, domain.GommageClassSystematic)
	}
}

func TestClassifyGommage_Mixed_NoMajority(t *testing.T) {
	reasons := []string{"timeout after 120s", "rate_limit: 429", "blocker: stuck"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassSystematic {
		t.Errorf("mixed with no majority should be systematic, got %q", got)
	}
}

func TestClassifyGommage_Empty(t *testing.T) {
	got := domain.ClassifyGommage(nil)
	if got != domain.GommageClassSystematic {
		t.Errorf("empty reasons should be systematic, got %q", got)
	}
}

func TestClassifyGommage_EvenSplit_FallsBackToSystematic(t *testing.T) {
	// 2 reasons: 1 timeout + 1 blocker → no strict majority → systematic
	reasons := []string{"timeout after 120s", "blocker: stuck"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassSystematic {
		t.Errorf("even split (1/2) should be systematic, got %q", got)
	}
}

func TestClassifyGommage_StrictMajority_ThreeOfFive(t *testing.T) {
	// 3 of 5 = strict majority
	reasons := []string{"timeout x", "timeout y", "timeout z", "blocker a", "rate_limit b"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassTimeout {
		t.Errorf("3/5 timeout should be timeout, got %q", got)
	}
}

func TestClassifyGommage_TwoOfFour_NotMajority(t *testing.T) {
	// 2 of 4 = exactly half, NOT majority → systematic
	reasons := []string{"timeout x", "timeout y", "blocker a", "blocker b"}
	got := domain.ClassifyGommage(reasons)
	if got != domain.GommageClassSystematic {
		t.Errorf("2/4 split should be systematic, got %q", got)
	}
}
