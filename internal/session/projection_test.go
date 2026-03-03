package session

import (
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func makeProjectionEvent(t EventType, data any) domain.Event {
	ev, err := domain.NewEvent(t, data, time.Now())
	if err != nil {
		panic(err)
	}
	return ev
}

func makeProjectionEventAt(t EventType, data any, ts time.Time) domain.Event {
	ev, err := domain.NewEvent(t, data, ts)
	if err != nil {
		panic(err)
	}
	return ev
}

type EventType = domain.EventType

func TestProjectState_Empty(t *testing.T) {
	state := ProjectState(nil)

	if state.TotalExpeditions != 0 {
		t.Errorf("TotalExpeditions = %d, want 0", state.TotalExpeditions)
	}
	if state.GradientLevel != 0 {
		t.Errorf("GradientLevel = %d, want 0", state.GradientLevel)
	}
}

func TestProjectState_ExpeditionCompleted(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionStarted, domain.ExpeditionStartedData{
			Expedition: 1, Worker: 0, Model: "sonnet",
		}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success",
		}),
		makeProjectionEvent(domain.EventExpeditionStarted, domain.ExpeditionStartedData{
			Expedition: 2, Worker: 0, Model: "sonnet",
		}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "failed",
		}),
	}

	state := ProjectState(events)

	if state.TotalExpeditions != 2 {
		t.Errorf("TotalExpeditions = %d, want 2", state.TotalExpeditions)
	}
	if state.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", state.Succeeded)
	}
	if state.Failed != 1 {
		t.Errorf("Failed = %d, want 1", state.Failed)
	}
	if state.LastExpedition != 2 {
		t.Errorf("LastExpedition = %d, want 2", state.LastExpedition)
	}
	if state.LastStatus != "failed" {
		t.Errorf("LastStatus = %q, want %q", state.LastStatus, "failed")
	}
}

func TestProjectState_SkippedExpedition(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "skipped",
		}),
	}

	state := ProjectState(events)

	if state.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1", state.Skipped)
	}
	if state.TotalExpeditions != 1 {
		t.Errorf("TotalExpeditions = %d, want 1", state.TotalExpeditions)
	}
}

func TestProjectState_GradientChanged(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventGradientChanged, domain.GradientChangedData{
			Level: 3, Operator: "auto",
		}),
		makeProjectionEvent(domain.EventGradientChanged, domain.GradientChangedData{
			Level: 5, Operator: "manual",
		}),
	}

	state := ProjectState(events)

	if state.GradientLevel != 5 {
		t.Errorf("GradientLevel = %d, want 5", state.GradientLevel)
	}
}

func TestProjectState_DMailCounts(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "spec-1"}),
		makeProjectionEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "spec-2"}),
		makeProjectionEvent(domain.EventInboxReceived, domain.InboxReceivedData{Name: "report-1", Severity: "info"}),
	}

	state := ProjectState(events)

	if state.DMailsStaged != 2 {
		t.Errorf("DMailsStaged = %d, want 2", state.DMailsStaged)
	}
	if state.InboxReceived != 1 {
		t.Errorf("InboxReceived = %d, want 1", state.InboxReceived)
	}
}

func TestProjectState_FullReplay(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionStarted, domain.ExpeditionStartedData{Expedition: 1}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 1, Status: "success"}),
		makeProjectionEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "spec-1"}),
		makeProjectionEvent(domain.EventDMailFlushed, domain.DMailFlushedData{Count: 1}),
		makeProjectionEvent(domain.EventGradientChanged, domain.GradientChangedData{Level: 2}),
		makeProjectionEvent(domain.EventExpeditionStarted, domain.ExpeditionStartedData{Expedition: 2}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 2, Status: "success"}),
		makeProjectionEvent(domain.EventInboxReceived, domain.InboxReceivedData{Name: "report-1"}),
	}

	state := ProjectState(events)

	if state.TotalExpeditions != 2 {
		t.Errorf("TotalExpeditions = %d, want 2", state.TotalExpeditions)
	}
	if state.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", state.Succeeded)
	}
	if state.GradientLevel != 2 {
		t.Errorf("GradientLevel = %d, want 2", state.GradientLevel)
	}
	if state.DMailsStaged != 1 {
		t.Errorf("DMailsStaged = %d, want 1", state.DMailsStaged)
	}
	if state.DMailsFlushed != 1 {
		t.Errorf("DMailsFlushed = %d, want 1", state.DMailsFlushed)
	}
	if state.InboxReceived != 1 {
		t.Errorf("InboxReceived = %d, want 1", state.InboxReceived)
	}
}

func TestProjectState_LastExpeditionAt(t *testing.T) {
	ts1 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 3, 1, 11, 30, 0, 0, time.UTC)

	events := []domain.Event{
		makeProjectionEventAt(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success",
		}, ts1),
		makeProjectionEventAt(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "failed",
		}, ts2),
	}

	state := ProjectState(events)

	if !state.LastExpeditionAt.Equal(ts2) {
		t.Errorf("LastExpeditionAt = %v, want %v", state.LastExpeditionAt, ts2)
	}
}

func TestProjectState_ErrorRate(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 1, Status: "success"}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 2, Status: "failed"}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 3, Status: "success"}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 4, Status: "failed"}),
	}

	state := ProjectState(events)

	got := state.ErrorRate()
	want := 0.5 // 2 failed out of 4
	if got != want {
		t.Errorf("ErrorRate() = %f, want %f", got, want)
	}
}

func TestProjectState_ErrorRate_Empty(t *testing.T) {
	state := ProjectState(nil)

	got := state.ErrorRate()
	if got != 0.0 {
		t.Errorf("ErrorRate() = %f, want 0.0 for empty state", got)
	}
}

func TestProjectState_GommageCount(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventGommageTriggered, domain.GommageTriggeredData{Expedition: 1, ConsecutiveFailures: 3}),
		makeProjectionEvent(domain.EventGommageTriggered, domain.GommageTriggeredData{Expedition: 5, ConsecutiveFailures: 3}),
	}

	state := ProjectState(events)

	if state.GommageCount != 2 {
		t.Errorf("GommageCount = %d, want 2", state.GommageCount)
	}
}

func TestProjectState_ConsecutiveFailures(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 1, Status: "failed"}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 2, Status: "failed"}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 3, Status: "success"}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 4, Status: "failed"}),
	}

	state := ProjectState(events)

	if state.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1 (reset by success at expedition 3)", state.ConsecutiveFailures)
	}
}

func TestProjectState_LastIssueID(t *testing.T) {
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success", IssueID: "PROJ-123",
		}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "success", IssueID: "PROJ-456",
		}),
	}

	state := ProjectState(events)

	if state.LastIssueID != "PROJ-456" {
		t.Errorf("LastIssueID = %q, want %q", state.LastIssueID, "PROJ-456")
	}
}

func TestProjectState_IgnoresUnknownEvents(t *testing.T) {
	events := []domain.Event{
		{ID: "x", Type: "unknown.event", Timestamp: time.Now(), Data: []byte(`{}`)},
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{Expedition: 1, Status: "success"}),
	}

	state := ProjectState(events)

	if state.TotalExpeditions != 1 {
		t.Errorf("TotalExpeditions = %d, want 1", state.TotalExpeditions)
	}
}
