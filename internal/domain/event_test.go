package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestValidateEvent_Valid(t *testing.T) {
	// given
	event := domain.Event{
		ID:        "test-001",
		Type:      domain.EventExpeditionStarted,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{"expedition":1}`),
	}

	// when
	err := domain.ValidateEvent(event)

	// then
	if err != nil {
		t.Errorf("expected no error for valid event, got %v", err)
	}
}

func TestValidateEvent_EmptyID(t *testing.T) {
	// given
	event := domain.Event{
		Type:      domain.EventExpeditionStarted,
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{}`),
	}

	// when
	err := domain.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for empty ID")
	}
}

func TestValidateEvent_EmptyType(t *testing.T) {
	// given
	event := domain.Event{
		ID:        "test-001",
		Timestamp: time.Now(),
		Data:      json.RawMessage(`{}`),
	}

	// when
	err := domain.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for empty Type")
	}
}

func TestValidateEvent_ZeroTimestamp(t *testing.T) {
	// given
	event := domain.Event{
		ID:   "test-001",
		Type: domain.EventExpeditionStarted,
		Data: json.RawMessage(`{}`),
	}

	// when
	err := domain.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for zero Timestamp")
	}
}

func TestValidateEvent_NilData(t *testing.T) {
	// given
	event := domain.Event{
		ID:        "test-001",
		Type:      domain.EventExpeditionStarted,
		Timestamp: time.Now(),
	}

	// when
	err := domain.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for nil Data")
	}
}

func TestValidateEvent_AllFieldsInvalid(t *testing.T) {
	// given
	event := domain.Event{}

	// when
	err := domain.ValidateEvent(event)

	// then
	if err == nil {
		t.Error("expected error for fully invalid event")
	}
}

func TestNewEvent_CreatesValidEvent(t *testing.T) {
	// given
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	payload := domain.ExpeditionStartedData{Expedition: 1, Worker: 0, Model: "opus"}

	// when
	event, err := domain.NewEvent(domain.EventExpeditionStarted, payload, now)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.ID == "" {
		t.Error("expected non-empty ID")
	}
	if event.Type != domain.EventExpeditionStarted {
		t.Errorf("Type = %q, want %q", event.Type, domain.EventExpeditionStarted)
	}
	if !event.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", event.Timestamp, now)
	}
	if err := domain.ValidateEvent(event); err != nil {
		t.Errorf("NewEvent produced invalid event: %v", err)
	}
}

func TestEventMarshalRoundTrip(t *testing.T) {
	// given
	now := time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC)
	event := domain.Event{
		ID:        "test-roundtrip",
		Type:      domain.EventExpeditionCompleted,
		Timestamp: now,
		Data:      json.RawMessage(`{"expedition":1,"status":"success"}`),
	}

	// when
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got domain.Event
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
	types := []domain.EventType{
		domain.EventExpeditionStarted,
		domain.EventExpeditionCompleted,
		domain.EventDMailStaged,
		domain.EventDMailFlushed,
		domain.EventDMailArchived,
		domain.EventGradientChanged,
		domain.EventGommageTriggered,
		domain.EventInboxReceived,
	}

	seen := make(map[domain.EventType]bool)
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

func TestEvent_CorrelationFields_Serialize(t *testing.T) {
	// given
	ev, err := domain.NewEvent("test.event", map[string]string{"k": "v"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ev.CorrelationID = "corr-123"
	ev.CausationID = "cause-456"

	// when
	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)

	// then
	if !strings.Contains(s, `"correlation_id":"corr-123"`) {
		t.Errorf("missing correlation_id: %s", s)
	}
	if !strings.Contains(s, `"causation_id":"cause-456"`) {
		t.Errorf("missing causation_id: %s", s)
	}
}

func TestEvent_CorrelationFields_OmitEmpty(t *testing.T) {
	// given
	ev, _ := domain.NewEvent("test.event", map[string]string{"k": "v"}, time.Now())

	// when
	data, _ := json.Marshal(ev)

	// then
	if strings.Contains(string(data), "correlation_id") {
		t.Errorf("empty CorrelationID should be omitted")
	}
}

func TestEvent_SchemaVersion_SetByNewEvent(t *testing.T) {
	// given / when
	ev, _ := domain.NewEvent("test.event", map[string]string{"k": "v"}, time.Now())

	// then
	if ev.SchemaVersion != domain.CurrentEventSchemaVersion {
		t.Errorf("got %d, want %d", ev.SchemaVersion, domain.CurrentEventSchemaVersion)
	}
}

func TestEvent_SchemaVersion_ZeroIsLegacy(t *testing.T) {
	// given
	raw := `{"id":"abc","type":"test","timestamp":"2026-01-01T00:00:00Z","data":{}}`

	// when
	var ev domain.Event
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}

	// then
	if ev.SchemaVersion != 0 {
		t.Errorf("legacy event should have SchemaVersion 0, got %d", ev.SchemaVersion)
	}
}

func TestValidateEvent_RejectsFutureSchema(t *testing.T) {
	// given
	ev, _ := domain.NewEvent("test.event", map[string]string{"k": "v"}, time.Now())
	ev.SchemaVersion = domain.CurrentEventSchemaVersion + 1

	// when
	err := domain.ValidateEvent(ev)

	// then
	if err == nil {
		t.Error("expected error for future schema version")
	}
}

func TestEventsDir(t *testing.T) {
	// when
	dir := domain.EventsDir("/repo")

	// then
	if dir != "/repo/.expedition/events" {
		t.Errorf("EventsDir = %q, want /repo/.expedition/events", dir)
	}
}
