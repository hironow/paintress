package session

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestStatus_EmptyState(t *testing.T) {
	// given — fresh directory with no events, no inbox, no archive
	baseDir := t.TempDir()

	// when
	report := Status(context.Background(), baseDir, &domain.NopLogger{})

	// then
	if report.Expeditions != 0 {
		t.Errorf("Expeditions = %d, want 0", report.Expeditions)
	}
	if report.Successes != 0 {
		t.Errorf("Successes = %d, want 0", report.Successes)
	}
	if report.Failures != 0 {
		t.Errorf("Failures = %d, want 0", report.Failures)
	}
	if report.SuccessRate != 0.0 {
		t.Errorf("SuccessRate = %f, want 0.0", report.SuccessRate)
	}
	if report.GradientLevel != 0 {
		t.Errorf("GradientLevel = %d, want 0", report.GradientLevel)
	}
	if report.InboxCount != 0 {
		t.Errorf("InboxCount = %d, want 0", report.InboxCount)
	}
	if report.ArchiveCount != 0 {
		t.Errorf("ArchiveCount = %d, want 0", report.ArchiveCount)
	}
	if !report.LastExpedition.IsZero() {
		t.Errorf("LastExpedition should be zero, got %v", report.LastExpedition)
	}
	if report.Continent != baseDir {
		t.Errorf("Continent = %q, want %q", report.Continent, baseDir)
	}
}

func TestStatus_WithMailDirs(t *testing.T) {
	// given — create inbox and archive with files
	baseDir := t.TempDir()
	inboxDir := domain.InboxDir(baseDir)
	archiveDir := domain.ArchiveDir(baseDir)

	if err := os.MkdirAll(inboxDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 2 inbox files
	for _, name := range []string{"report-proj-1.md", "report-proj-2.md"} {
		if err := os.WriteFile(filepath.Join(inboxDir, name), []byte("---\nname: test\nkind: report\ndescription: test\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create 3 archive files
	for _, name := range []string{"report-a1.md", "report-a2.md", "report-a3.md"} {
		if err := os.WriteFile(filepath.Join(archiveDir, name), []byte("---\nname: test\nkind: report\ndescription: test\n---\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// when
	report := Status(context.Background(), baseDir, &domain.NopLogger{})

	// then
	if report.InboxCount != 2 {
		t.Errorf("InboxCount = %d, want 2", report.InboxCount)
	}
	if report.ArchiveCount != 3 {
		t.Errorf("ArchiveCount = %d, want 3", report.ArchiveCount)
	}
}

func TestStatus_WithEvents(t *testing.T) {
	// given — create event store with expedition events
	baseDir := t.TempDir()
	stateDir := filepath.Join(baseDir, ".expedition")
	store := NewEventStore(stateDir, &domain.NopLogger{})

	ts := time.Date(2026, 3, 2, 10, 0, 0, 0, time.UTC)

	events := []domain.Event{
		makeStatusEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success", IssueID: "PROJ-1",
		}, ts),
		makeStatusEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "success", IssueID: "PROJ-2",
		}, ts.Add(time.Minute)),
		makeStatusEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 3, Status: "failed", IssueID: "PROJ-3",
		}, ts.Add(2*time.Minute)),
		makeStatusEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 4, Status: "skipped", IssueID: "PROJ-4",
		}, ts.Add(3*time.Minute)),
		makeStatusEvent(domain.EventGradientChanged, domain.GradientChangedData{
			Level: 3, Operator: "charge",
		}, ts.Add(4*time.Minute)),
	}

	if _, err := store.Append(events...); err != nil {
		t.Fatal(err)
	}

	// when
	report := Status(context.Background(), baseDir, &domain.NopLogger{})

	// then
	if report.Expeditions != 4 {
		t.Errorf("Expeditions = %d, want 4", report.Expeditions)
	}
	if report.Successes != 2 {
		t.Errorf("Successes = %d, want 2", report.Successes)
	}
	if report.Failures != 1 {
		t.Errorf("Failures = %d, want 1", report.Failures)
	}
	// SuccessRate excludes skipped: 2 success / 3 (success+failed)
	wantRate := 2.0 / 3.0
	if report.SuccessRate < wantRate-0.01 || report.SuccessRate > wantRate+0.01 {
		t.Errorf("SuccessRate = %f, want ~%f", report.SuccessRate, wantRate)
	}
	if report.GradientLevel != 3 {
		t.Errorf("GradientLevel = %d, want 3", report.GradientLevel)
	}
	if !report.LastExpedition.Equal(ts.Add(3 * time.Minute)) {
		t.Errorf("LastExpedition = %v, want %v", report.LastExpedition, ts.Add(3*time.Minute))
	}
}

func makeStatusEvent(eventType domain.EventType, data any, ts time.Time) domain.Event {
	ev, err := domain.NewEvent(eventType, data, ts)
	if err != nil {
		panic(err)
	}
	return ev
}
