package eventsource

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hironow/paintress"
)

func TestFileEventStore_AppendAndLoadAll(t *testing.T) {
	// given
	dir := t.TempDir()
	store := NewFileEventStore(dir)
	now := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	ev, err := paintress.NewEvent(paintress.EventExpeditionStarted,
		paintress.ExpeditionStartedData{Expedition: 1, Worker: 0, Model: "opus"}, now)
	if err != nil {
		t.Fatal(err)
	}

	// when
	if err := store.Append(ev); err != nil {
		t.Fatalf("Append: %v", err)
	}
	events, err := store.LoadAll()

	// then
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("len = %d, want 1", len(events))
	}
	if events[0].Type != paintress.EventExpeditionStarted {
		t.Errorf("Type = %q, want %q", events[0].Type, paintress.EventExpeditionStarted)
	}
}

func TestFileEventStore_LoadAll_EmptyDir(t *testing.T) {
	// given: directory does not exist
	dir := filepath.Join(t.TempDir(), "nonexistent")
	store := NewFileEventStore(dir)

	// when
	events, err := store.LoadAll()

	// then
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if events != nil {
		t.Errorf("expected nil for missing directory, got %v", events)
	}
}

func TestFileEventStore_DailyRotation(t *testing.T) {
	// given: events on two different days
	dir := t.TempDir()
	store := NewFileEventStore(dir)
	day1 := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)

	ev1, _ := paintress.NewEvent(paintress.EventExpeditionStarted,
		paintress.ExpeditionStartedData{Expedition: 1}, day1)
	ev2, _ := paintress.NewEvent(paintress.EventExpeditionCompleted,
		paintress.ExpeditionCompletedData{Expedition: 1, Status: "success"}, day2)

	// when
	if err := store.Append(ev1, ev2); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// then: two separate daily files
	entries, _ := os.ReadDir(dir)
	jsonlCount := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".jsonl" {
			jsonlCount++
		}
	}
	if jsonlCount != 2 {
		t.Errorf("expected 2 .jsonl files, got %d", jsonlCount)
	}

	// LoadAll returns both events in chronological order
	events, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len = %d, want 2", len(events))
	}
	if events[0].Timestamp.After(events[1].Timestamp) {
		t.Error("events not in chronological order")
	}
}

func TestFileEventStore_LoadSince(t *testing.T) {
	// given
	dir := t.TempDir()
	store := NewFileEventStore(dir)
	t1 := time.Date(2026, 2, 27, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 27, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

	ev1, _ := paintress.NewEvent(paintress.EventExpeditionStarted,
		paintress.ExpeditionStartedData{Expedition: 1}, t1)
	ev2, _ := paintress.NewEvent(paintress.EventExpeditionCompleted,
		paintress.ExpeditionCompletedData{Expedition: 1, Status: "success"}, t2)
	ev3, _ := paintress.NewEvent(paintress.EventExpeditionStarted,
		paintress.ExpeditionStartedData{Expedition: 2}, t3)

	store.Append(ev1, ev2, ev3)

	// when: load events after t1
	events, err := store.LoadSince(t1)

	// then: only ev2 and ev3
	if err != nil {
		t.Fatalf("LoadSince: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len = %d, want 2", len(events))
	}
	if events[0].Type != paintress.EventExpeditionCompleted {
		t.Errorf("events[0].Type = %q, want expedition.completed", events[0].Type)
	}
}

func TestFileEventStore_Append_RejectsInvalidEvent(t *testing.T) {
	// given
	dir := t.TempDir()
	store := NewFileEventStore(dir)
	invalid := paintress.Event{} // all fields empty

	// when
	err := store.Append(invalid)

	// then
	if err == nil {
		t.Error("expected error for invalid event")
	}
}

func TestFileEventStore_StableOrderForSameTimestamp(t *testing.T) {
	// given: two events with identical timestamps
	dir := t.TempDir()
	store := NewFileEventStore(dir)
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)

	ev1, _ := paintress.NewEvent(paintress.EventDMailStaged,
		paintress.DMailStagedData{Name: "first"}, now)
	ev2, _ := paintress.NewEvent(paintress.EventDMailFlushed,
		paintress.DMailFlushedData{Count: 1}, now)

	store.Append(ev1, ev2)

	// when
	events, err := store.LoadAll()

	// then: stable sort preserves insertion order
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("len = %d, want 2", len(events))
	}
	if events[0].Type != paintress.EventDMailStaged {
		t.Errorf("first event Type = %q, want dmail.staged", events[0].Type)
	}
	if events[1].Type != paintress.EventDMailFlushed {
		t.Errorf("second event Type = %q, want dmail.flushed", events[1].Type)
	}
}
