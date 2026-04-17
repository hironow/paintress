package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	// nosemgrep: clean-architecture.domain-imports-usecase-go -- uuid is a pure ID library with no infrastructure coupling; domain identity requires it [permanent]
	"github.com/google/uuid"
)

// nosemgrep: clean-architecture.sql-in-domain-go -- word 'update' in comment is English prose, not SQL; regex false positive [permanent]
// EventApplier applies domain events to update materialized projections.
// Note: context is intentionally absent — domain types must remain pure.
// Use ContextEventApplier (usecase/port) when caller context is required.
type EventApplier interface {
	Apply(event Event) error
	Rebuild(events []Event) error
	Serialize() ([]byte, error)
	Deserialize(data []byte) error
}

// EventType identifies the kind of domain event.
type EventType string

const (
	EventExpeditionStarted    EventType = "expedition.started"
	EventExpeditionCompleted  EventType = "expedition.completed"
	EventDMailStaged          EventType = "dmail.staged"
	EventDMailFlushed         EventType = "dmail.flushed"
	EventDMailArchived        EventType = "dmail.archived"
	EventGradientChanged      EventType = "gradient.changed"
	EventGommageTriggered     EventType = "gommage.triggered"
	EventGommageRecovery      EventType = "gommage.recovery"
	EventExpeditionCheckpoint EventType = "expedition.checkpoint"
	EventInboxReceived        EventType = "inbox.received"
	EventRetryAttempted       EventType = "retry.attempted"
	EventEscalated            EventType = "issue.escalated"
	EventResolved             EventType = "issue.resolved"
	EventSpecRegistered       EventType = "spec.registered"
	EventSystemCutover        EventType = "system.cutover"
)

// validEventTypes is the set of recognized EventType values.
var validEventTypes = map[EventType]bool{
	EventExpeditionStarted:    true,
	EventExpeditionCompleted:  true,
	EventDMailStaged:          true,
	EventDMailFlushed:         true,
	EventDMailArchived:        true,
	EventGradientChanged:      true,
	EventGommageTriggered:     true,
	EventGommageRecovery:      true,
	EventExpeditionCheckpoint: true,
	EventInboxReceived:        true,
	EventRetryAttempted:       true,
	EventEscalated:            true,
	EventResolved:             true,
	EventSpecRegistered:       true,
	EventSystemCutover:        true,
}

// ValidEventType returns true if the given EventType is recognized.
func ValidEventType(t EventType) bool {
	return validEventTypes[t]
}

// AllValidEventTypes returns a copy of the canonical event type set (for testing).
func AllValidEventTypes() map[EventType]bool {
	cp := make(map[EventType]bool, len(validEventTypes))
	for k, v := range validEventTypes {
		cp[k] = v
	}
	return cp
}

// CurrentEventSchemaVersion is the schema version set by NewEvent.
// Version 0 represents pre-Phase2 legacy events.
const CurrentEventSchemaVersion uint8 = 1

// Event is the envelope for all domain events in the event store.
type Event struct {
	SchemaVersion uint8           `json:"schema_version,omitempty"`
	ID            string          `json:"id"`
	Type          EventType       `json:"type"`
	Timestamp     time.Time       `json:"timestamp"`
	Data          json.RawMessage `json:"data"`
	SessionID     string          `json:"session_id,omitempty"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	CausationID   string          `json:"causation_id,omitempty"`
	AggregateID   string          `json:"aggregate_id,omitempty"`
	AggregateType string          `json:"aggregate_type,omitempty"`
	SeqNr         uint64          `json:"seq_nr,omitempty"`
}

// ValidateEvent checks that an Event has all required fields populated.
func ValidateEvent(e Event) error {
	var errs []string
	if e.ID == "" {
		errs = append(errs, "ID is required")
	}
	if e.Type == "" {
		errs = append(errs, "Type is required")
	} else if !ValidEventType(e.Type) {
		errs = append(errs, fmt.Sprintf("Type %q is not a recognized event type", e.Type))
	}
	if e.Timestamp.IsZero() {
		errs = append(errs, "Timestamp must not be zero")
	}
	if len(e.Data) == 0 {
		errs = append(errs, "Data must not be empty")
	}
	if e.SchemaVersion > CurrentEventSchemaVersion {
		errs = append(errs, fmt.Sprintf("schema_version %d exceeds supported version %d", e.SchemaVersion, CurrentEventSchemaVersion))
	}
	if len(errs) > 0 {
		return errors.New("invalid event: " + strings.Join(errs, "; "))
	}
	return nil
}

// NewEvent creates a new Event with a UUID, the given timestamp, and marshaled data payload.
func NewEvent(eventType EventType, data any, timestamp time.Time) (Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return Event{}, fmt.Errorf("marshal event data: %w", err)
	}
	return Event{
		SchemaVersion: CurrentEventSchemaVersion,
		ID:            uuid.NewString(),
		Type:          eventType,
		Timestamp:     timestamp,
		Data:          raw,
	}, nil
}

// AppendResult captures metrics from an event store Append operation.
type AppendResult struct {
	BytesWritten int // total bytes written to event files
}

// LoadResult captures metrics from an event store Load operation.
type LoadResult struct {
	FileCount        int // number of .jsonl files scanned
	CorruptLineCount int // number of lines skipped due to parse errors
}

// EventsDir returns the path to the events directory under .expedition/.
func EventsDir(continent string) string {
	return filepath.Join(continent, StateDir, "events")
}

// --- Event payload types ---

// ExpeditionStartedData is the payload for EventExpeditionStarted.
type ExpeditionStartedData struct {
	Expedition int    `json:"expedition"`
	Worker     int    `json:"worker"`
	Model      string `json:"model"`
}

// ExpeditionCompletedData is the payload for EventExpeditionCompleted.
type ExpeditionCompletedData struct {
	Expedition int    `json:"expedition"`
	Status     string `json:"status"`
	IssueID    string `json:"issue_id,omitempty"`
	WaveID     string `json:"wave_id,omitempty"` // explicit wave reference for Read Model
	StepID     string `json:"step_id,omitempty"` // explicit step reference for Read Model
	BugsFound  string `json:"bugs_found,omitempty"`
}

// DMailStagedData is the payload for EventDMailStaged.
type DMailStagedData struct {
	Name string `json:"name"`
}

// DMailFlushedData is the payload for EventDMailFlushed.
type DMailFlushedData struct {
	Count int `json:"count"`
}

// DMailArchivedData is the payload for EventDMailArchived.
type DMailArchivedData struct {
	Name string `json:"name"`
}

// GradientChangedData is the payload for EventGradientChanged.
type GradientChangedData struct {
	Level    int    `json:"level"`
	Operator string `json:"operator"`
}

// GommageTriggeredData is the payload for EventGommageTriggered.
type GommageTriggeredData struct {
	Expedition          int          `json:"expedition"`
	ConsecutiveFailures int          `json:"consecutive_failures"`
	Class               GommageClass `json:"class,omitempty"`
	RecoveryAction      string       `json:"recovery_action,omitempty"`
	RetryNum            int          `json:"retry_num,omitempty"`
}

// GommageRecoveryData is the payload for EventGommageRecovery.
type GommageRecoveryData struct {
	Expedition int          `json:"expedition"`
	Class      GommageClass `json:"class"`
	Action     string       `json:"action"`
	RetryNum   int          `json:"retry_num"`
	Cooldown   string       `json:"cooldown"`
}

// ExpeditionCheckpointData is the payload for EventExpeditionCheckpoint.
type ExpeditionCheckpointData struct {
	Expedition  int    `json:"expedition"`
	Phase       string `json:"phase"`
	WorkDir     string `json:"work_dir"`
	CommitCount int    `json:"commit_count"`
}

// InboxReceivedData is the payload for EventInboxReceived.
type InboxReceivedData struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
}

// RetryAttemptedData is the payload for EventRetryAttempted.
type RetryAttemptedData struct {
	DMail   string `json:"dmail"`
	Attempt int    `json:"attempt"`
}

// EscalatedData is the payload for EventEscalated.
type EscalatedData struct {
	DMail  string   `json:"dmail"`
	Issues []string `json:"issues"`
}

// ResolvedData is the payload for EventResolved.
type ResolvedData struct {
	DMail  string   `json:"dmail"`
	Issues []string `json:"issues"`
}

// SpecRegisteredData is the payload for EventSpecRegistered.
// Records a wave specification from a D-Mail so that wave/step definitions
// persist in the event store after the D-Mail is archived.
type SpecRegisteredData struct {
	WaveID string        `json:"wave_id"`
	Steps  []WaveStepDef `json:"steps"`
	Source string        `json:"source"` // D-Mail name for tracing
}
