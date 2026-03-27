package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EventApplier applies domain events to update materialized projections.
type EventApplier interface {
	Apply(event Event) error
	Rebuild(events []Event) error
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
	EventEscalated            EventType = "escalated"
	EventResolved             EventType = "resolved"
)

// Event is the envelope for all domain events in the event store.
type Event struct {
	ID            string          `json:"id"`
	Type          EventType       `json:"type"`
	Timestamp     time.Time       `json:"timestamp"`
	Data          json.RawMessage `json:"data"`
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
	}
	if e.Timestamp.IsZero() {
		errs = append(errs, "Timestamp must not be zero")
	}
	if len(e.Data) == 0 {
		errs = append(errs, "Data must not be empty")
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
		ID:        uuid.NewString(),
		Type:      eventType,
		Timestamp: timestamp,
		Data:      raw,
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
