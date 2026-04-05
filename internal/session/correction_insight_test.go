package session_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestWriteCorrectionInsight_AppendsImprovementInsight(t *testing.T) {
	base := t.TempDir()
	if err := os.MkdirAll(filepath.Join(base, ".expedition", "insights"), 0o755); err != nil {
		t.Fatalf("mkdir insights: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, ".expedition", ".run"), 0o755); err != nil {
		t.Fatalf("mkdir run: %v", err)
	}
	w := session.NewInsightWriter(filepath.Join(base, ".expedition", "insights"), filepath.Join(base, ".expedition", ".run"))
	mail := domain.DMail{
		Name: "feedback-1",
		Metadata: map[string]string{
			domain.MetadataFailureType:      string(domain.FailureTypeExecutionFailure),
			domain.MetadataSeverity:         "medium",
			domain.MetadataTargetAgent:      "paintress",
			domain.MetadataCorrectiveAction: "retry",
			domain.MetadataOutcome:          string(domain.ImprovementOutcomePending),
		},
	}

	session.WriteCorrectionInsight(w, mail, &domain.NopLogger{})

	file, err := w.Read("improvement-loop.md")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(file.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(file.Entries))
	}
	if file.Entries[0].Title != "feedback-1" {
		t.Fatalf("title = %q, want feedback-1", file.Entries[0].Title)
	}
}
