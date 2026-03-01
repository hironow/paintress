package session

import (
	"testing"
	"time"

	"github.com/hironow/paintress"
)

func makeProjectionEvent(t EventType, data any) paintress.Event {
	ev, err := paintress.NewEvent(t, data, time.Now())
	if err != nil {
		panic(err)
	}
	return ev
}

func makeProjectionEventAt(t EventType, data any, ts time.Time) paintress.Event {
	ev, err := paintress.NewEvent(t, data, ts)
	if err != nil {
		panic(err)
	}
	return ev
}

type EventType = paintress.EventType

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
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventExpeditionStarted, paintress.ExpeditionStartedData{
			Expedition: 1, Worker: 0, Model: "sonnet",
		}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
			Expedition: 1, Status: "success",
		}),
		makeProjectionEvent(paintress.EventExpeditionStarted, paintress.ExpeditionStartedData{
			Expedition: 2, Worker: 0, Model: "sonnet",
		}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
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
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
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
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventGradientChanged, paintress.GradientChangedData{
			Level: 3, Operator: "auto",
		}),
		makeProjectionEvent(paintress.EventGradientChanged, paintress.GradientChangedData{
			Level: 5, Operator: "manual",
		}),
	}

	state := ProjectState(events)

	if state.GradientLevel != 5 {
		t.Errorf("GradientLevel = %d, want 5", state.GradientLevel)
	}
}

func TestProjectState_DMailCounts(t *testing.T) {
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventDMailStaged, paintress.DMailStagedData{Name: "spec-1"}),
		makeProjectionEvent(paintress.EventDMailStaged, paintress.DMailStagedData{Name: "spec-2"}),
		makeProjectionEvent(paintress.EventInboxReceived, paintress.InboxReceivedData{Name: "report-1", Severity: "info"}),
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
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventExpeditionStarted, paintress.ExpeditionStartedData{Expedition: 1}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 1, Status: "success"}),
		makeProjectionEvent(paintress.EventDMailStaged, paintress.DMailStagedData{Name: "spec-1"}),
		makeProjectionEvent(paintress.EventDMailFlushed, paintress.DMailFlushedData{Count: 1}),
		makeProjectionEvent(paintress.EventGradientChanged, paintress.GradientChangedData{Level: 2}),
		makeProjectionEvent(paintress.EventExpeditionStarted, paintress.ExpeditionStartedData{Expedition: 2}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 2, Status: "success"}),
		makeProjectionEvent(paintress.EventInboxReceived, paintress.InboxReceivedData{Name: "report-1"}),
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

	events := []paintress.Event{
		makeProjectionEventAt(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
			Expedition: 1, Status: "success",
		}, ts1),
		makeProjectionEventAt(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
			Expedition: 2, Status: "failed",
		}, ts2),
	}

	state := ProjectState(events)

	if !state.LastExpeditionAt.Equal(ts2) {
		t.Errorf("LastExpeditionAt = %v, want %v", state.LastExpeditionAt, ts2)
	}
}

func TestProjectState_ErrorRate(t *testing.T) {
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 1, Status: "success"}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 2, Status: "failed"}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 3, Status: "success"}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 4, Status: "failed"}),
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
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventGommageTriggered, paintress.GommageTriggeredData{Expedition: 1, ConsecutiveFailures: 3}),
		makeProjectionEvent(paintress.EventGommageTriggered, paintress.GommageTriggeredData{Expedition: 5, ConsecutiveFailures: 3}),
	}

	state := ProjectState(events)

	if state.GommageCount != 2 {
		t.Errorf("GommageCount = %d, want 2", state.GommageCount)
	}
}

func TestProjectState_ConsecutiveFailures(t *testing.T) {
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 1, Status: "failed"}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 2, Status: "failed"}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 3, Status: "success"}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 4, Status: "failed"}),
	}

	state := ProjectState(events)

	if state.ConsecutiveFailures != 1 {
		t.Errorf("ConsecutiveFailures = %d, want 1 (reset by success at expedition 3)", state.ConsecutiveFailures)
	}
}

func TestProjectState_LastIssueID(t *testing.T) {
	events := []paintress.Event{
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
			Expedition: 1, Status: "success", IssueID: "PROJ-123",
		}),
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{
			Expedition: 2, Status: "success", IssueID: "PROJ-456",
		}),
	}

	state := ProjectState(events)

	if state.LastIssueID != "PROJ-456" {
		t.Errorf("LastIssueID = %q, want %q", state.LastIssueID, "PROJ-456")
	}
}

func TestProjectState_IgnoresUnknownEvents(t *testing.T) {
	events := []paintress.Event{
		{ID: "x", Type: "unknown.event", Timestamp: time.Now(), Data: []byte(`{}`)},
		makeProjectionEvent(paintress.EventExpeditionCompleted, paintress.ExpeditionCompletedData{Expedition: 1, Status: "success"}),
	}

	state := ProjectState(events)

	if state.TotalExpeditions != 1 {
		t.Errorf("TotalExpeditions = %d, want 1", state.TotalExpeditions)
	}
}
