package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestFileHandoverWriter_WritesMarkdown(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".expedition")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	w := &FileHandoverWriter{}
	state := domain.HandoverState{
		Tool:       "paintress",
		Operation:  "expedition",
		Timestamp:  time.Date(2026, 3, 14, 15, 30, 45, 0, time.UTC),
		InProgress: "Issue: TAP-123 Fix authentication flow",
		Completed:  []string{"Expedition #1: Applied (PR #45 merged)"},
		Remaining:  []string{"Expedition #4-5: Not started"},
		PartialState: map[string]string{
			"Branch": "feature/tap-123-fix-auth",
		},
	}

	err := w.WriteHandover(context.Background(), stateDir, state)
	if err != nil {
		t.Fatalf("WriteHandover: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(stateDir, "handover.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"# Handover",
		"INTERRUPTED",
		"TAP-123",
		"Expedition #1: Applied (PR #45 merged)",
		"Expedition #4-5: Not started",
		"Branch",
		"feature/tap-123-fix-auth",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("handover.md missing %q", want)
		}
	}
}

func TestFileHandoverWriter_OverwritesPrevious(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".expedition")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	w := &FileHandoverWriter{}
	first := domain.HandoverState{
		Tool: "paintress", Operation: "expedition",
		Timestamp: time.Now(), InProgress: "first",
	}
	second := domain.HandoverState{
		Tool: "paintress", Operation: "expedition",
		Timestamp: time.Now(), InProgress: "second",
	}

	if err := w.WriteHandover(context.Background(), stateDir, first); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteHandover(context.Background(), stateDir, second); err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(filepath.Join(stateDir, "handover.md"))
	content := string(data)
	if strings.Contains(content, "first") {
		t.Error("expected previous handover to be overwritten")
	}
	if !strings.Contains(content, "second") {
		t.Error("expected new handover content")
	}
}

func TestFileHandoverWriter_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dir := t.TempDir()
	w := &FileHandoverWriter{}
	state := domain.HandoverState{
		Tool: "paintress", Operation: "expedition", Timestamp: time.Now(),
	}

	err := w.WriteHandover(ctx, dir, state)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}
