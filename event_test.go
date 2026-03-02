package paintress_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hironow/paintress"
)

func TestValidateEvent_Valid(t *testing.T) {
	// given
	event := paintress.Event{
		ID:        "test-001",
		Type:      paintress.EventExpeditionStarted,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"expedition":1}`),
	}

	// when
	err := paintress.ValidateEvent(event)

	// then
	if err != nil {
		t.Errorf("expected no error for valid event, got %v", err)
	}
}

func TestValidateEvent_EmptyID(t *testing.T) {
	// given
	event := paintress.Event{
		Type:      paintress.EventExpeditionStarted,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{}`),
	}

	// when
	err := paintress.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestValidateEvent_EmptyType(t *testing.T) {
	// given
	event := paintress.Event{
		ID:        "test-001",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{}`),
	}

	// when
	err := paintress.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for empty Type")
	}
}

func TestValidateEvent_ZeroTimestamp(t *testing.T) {
	// given
	event := paintress.Event{
		ID:   "test-001",
		Type: paintress.EventExpeditionStarted,
		Data: json.RawMessage(`{}`),
	}

	// when
	err := paintress.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for zero Timestamp")
	}
}

func TestValidateEvent_NilData(t *testing.T) {
	// given
	event := paintress.Event{
		ID:        "test-001",
		Type:      paintress.EventExpeditionStarted,
		Timestamp: time.Now(),
	}

	// when
	err := paintress.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for nil Data")
	}
}

func TestValidateEvent_AllFieldsInvalid(t *testing.T) {
	// given
	event := paintress.Event{}

	// when
	err := paintress.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for fully invalid event")
	}
}

func TestNewEvent_CreatesValidEvent(t *testing.T) {
	// given
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	payload := paintress.ExpeditionStartedData{Expedition: 1, Worker: 0, Model: "opus"}

	// when
	event, err := paintress.NewEvent(paintress.EventExpeditionStarted, payload, now)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID == "" {
		t.Error("expected non-empty ID")
	}
	if event.Type != paintress.EventExpeditionStarted {
		t.Errorf("Type = %q, want %q", event.Type, paintress.EventExpeditionStarted)
	}
	if !event.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", event.Timestamp, now)
	}
	if err := paintress.ValidateEvent(event); err != nil {
		t.Errorf("NewEvent produced invalid event: %v", err)
	}
}

func TestEventMarshalRoundTrip(t *testing.T) {
	// given
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	event := paintress.Event{
		ID:        "test-roundtrip",
		Type:      paintress.EventExpeditionCompleted,
		Timestamp: now,
		Data:      json.RawMessage(`{"expedition":1,"status":"success"}`),
	}

	// when
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got paintress.Event
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// then
	if got.ID != event.ID {
		t.Errorf("ID = %q, want %q", got.ID, event.ID)
	}
	if got.Type != event.Type {
		t.Errorf("Type = %q, want %q", got.Type, event.Type)
	}
	if !got.Timestamp.Equal(event.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, event.Timestamp)
	}
}

func TestEventTypeConstants_Distinct(t *testing.T) {
	// then: all event types are distinct non-empty strings
	types := []paintress.EventType{
		paintress.EventExpeditionStarted,
		paintress.EventExpeditionCompleted,
		paintress.EventDMailStaged,
		paintress.EventDMailFlushed,
		paintress.EventDMailArchived,
		paintress.EventGradientChanged,
		paintress.EventGommageTriggered,
		paintress.EventInboxReceived,
	}

	seen := make(map[paintress.EventType]bool)
	for _, et := range types {
		if et == "" {
			t.Error("found empty EventType constant")
		}
		if seen[et] {
			t.Errorf("duplicate EventType: %q", et)
		}
		seen[et] = true
	}
}

func TestEventsDir(t *testing.T) {
	// when
	dir := paintress.EventsDir("/repo")

	// then
	if dir != "/repo/.expedition/events" {
		t.Errorf("EventsDir = %q, want /repo/.expedition/events", dir)
	}
}
