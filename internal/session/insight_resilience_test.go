package session_test

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestWriteLuminaInsights_FailureDoesNotPanic(t *testing.T) {
	// given — writer with non-existent directories (writes will fail)
	w := session.NewInsightWriter("/nonexistent/insights", "/nonexistent/run")
	luminas := []domain.Lumina{
		{Pattern: "test", Source: "failure-pattern", Uses: 2},
	}

	// when / then — should not panic or return error (best-effort)
	session.WriteLuminaInsights(w, luminas)
}

func TestWriteGommageInsight_FailureDoesNotPanic(t *testing.T) {
	// given — writer with non-existent directories (writes will fail)
	w := session.NewInsightWriter("/nonexistent/insights", "/nonexistent/run")

	// when / then — should not panic (best-effort)
	session.WriteGommageInsight(w, 5, 3, "/nonexistent")
}
