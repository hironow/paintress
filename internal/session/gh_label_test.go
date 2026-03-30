package session_test

import (
	"context"
	"testing"

	"github.com/hironow/paintress/internal/session"
)

func TestApplyIssueLabel_RejectsEmptyIssueID(t *testing.T) {
	// given
	issueID := ""
	label := "paintress:pr-open"

	// when
	err := session.ApplyIssueLabel(context.Background(), issueID, label, nil)

	// then
	if err == nil {
		t.Fatal("expected error for empty issue ID")
	}
}

func TestApplyIssueLabel_RejectsEmptyLabel(t *testing.T) {
	// given
	issueID := "42"
	label := ""

	// when
	err := session.ApplyIssueLabel(context.Background(), issueID, label, nil)

	// then
	if err == nil {
		t.Fatal("expected error for empty label")
	}
}
