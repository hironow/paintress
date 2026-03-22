package domain_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

// TestExtractReviewComments_EmptyOutput verifies empty output produces no comments.
func TestExtractReviewComments_EmptyOutput(t *testing.T) {
	// given
	output := ""

	// when
	comments := domain.ExtractReviewComments(output)

	// then
	if len(comments) != 0 {
		t.Errorf("ExtractReviewComments(empty) = %v, want empty slice", comments)
	}
}

// TestExtractReviewComments_NoPriorityTags verifies output without tags returns raw fallback.
func TestExtractReviewComments_NoPriorityTags(t *testing.T) {
	// given: output with "Review comment" keyword but no priority tags
	output := "Review comment: please add error handling"

	// when
	comments := domain.ExtractReviewComments(output)

	// then: should fall back to raw output as a single comment
	if len(comments) != 1 {
		t.Errorf("ExtractReviewComments without tags = %d comments, want 1 (raw fallback)", len(comments))
	}
}

// TestExtractReviewComments_SingleP0Tag verifies a P0 tag is extracted correctly.
func TestExtractReviewComments_SingleP0Tag(t *testing.T) {
	// given
	output := "[P0] Critical: nil pointer dereference in handler"

	// when
	comments := domain.ExtractReviewComments(output)

	// then
	if len(comments) != 1 {
		t.Fatalf("ExtractReviewComments = %d comments, want 1", len(comments))
	}
	if comments[0].Priority != 0 {
		t.Errorf("comment priority = %d, want 0 (P0)", comments[0].Priority)
	}
	if comments[0].Text == "" {
		t.Error("comment text should not be empty")
	}
}

// TestExtractReviewComments_MultipleTagsSortedByPriority verifies sort order (P0 first).
func TestExtractReviewComments_MultipleTagsSortedByPriority(t *testing.T) {
	// given: tags in reverse priority order
	output := "[P3] Style issue\n[P1] Missing test\n[P0] Critical bug\n[P2] Performance"

	// when
	comments := domain.ExtractReviewComments(output)

	// then: sorted so P0 first, then P1, P2, P3
	if len(comments) != 4 {
		t.Fatalf("ExtractReviewComments = %d comments, want 4", len(comments))
	}
	if comments[0].Priority != 0 {
		t.Errorf("first comment priority = %d, want 0 (P0 should be first)", comments[0].Priority)
	}
	if comments[1].Priority != 1 {
		t.Errorf("second comment priority = %d, want 1 (P1)", comments[1].Priority)
	}
	if comments[2].Priority != 2 {
		t.Errorf("third comment priority = %d, want 2 (P2)", comments[2].Priority)
	}
	if comments[3].Priority != 3 {
		t.Errorf("fourth comment priority = %d, want 3 (P3)", comments[3].Priority)
	}
}

// TestExtractReviewComments_AllPriorityLevels verifies all P0-P4 are detected.
func TestExtractReviewComments_AllPriorityLevels(t *testing.T) {
	// given
	output := "[P0] Bug\n[P1] Warning\n[P2] Style\n[P3] Suggestion\n[P4] Nitpick"

	// when
	comments := domain.ExtractReviewComments(output)

	// then
	if len(comments) != 5 {
		t.Fatalf("ExtractReviewComments = %d comments, want 5", len(comments))
	}
	for i, c := range comments {
		if c.Priority != i {
			t.Errorf("comments[%d].Priority = %d, want %d", i, c.Priority, i)
		}
	}
}

// TestReviewComment_HasPriorityAndText verifies ReviewComment struct fields.
func TestReviewComment_HasPriorityAndText(t *testing.T) {
	// given
	output := "[P1] Missing nil check in getUserHandler"

	// when
	comments := domain.ExtractReviewComments(output)

	// then
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Priority != 1 {
		t.Errorf("Priority = %d, want 1", comments[0].Priority)
	}
	if comments[0].Text == "" {
		t.Error("Text should not be empty")
	}
}

// TestExtractReviewComments_FallbackToRawWhenNoTagsAndNoKeyword verifies no-output case.
func TestExtractReviewComments_FallbackToRawWhenNoTagsAndNoKeyword(t *testing.T) {
	// given: plain error output, no tags or keyword
	output := "exit status 1"

	// when
	comments := domain.ExtractReviewComments(output)

	// then: no recognizable content → empty
	if len(comments) != 0 {
		t.Errorf("ExtractReviewComments(plain exit status) = %d comments, want 0", len(comments))
	}
}
