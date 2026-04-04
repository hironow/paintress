package session

import (
	"encoding/json"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// ExpeditionState is the materialized READ MODEL projected from events.
type ExpeditionState struct {
	TotalExpeditions    int       `json:"total_expeditions"`
	Succeeded           int       `json:"succeeded"`
	Failed              int       `json:"failed"`
	Skipped             int       `json:"skipped"`
	LastExpedition      int       `json:"last_expedition"`
	LastStatus          string    `json:"last_status"`
	LastExpeditionAt    time.Time `json:"last_expedition_at"`
	LastIssueID         string    `json:"last_issue_id"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	GommageCount        int       `json:"gommage_count"`
	GradientLevel       int       `json:"gradient_level"`
	DMailsStaged        int       `json:"dmails_staged"`
	DMailsFlushed       int       `json:"dmails_flushed"`
	InboxReceived       int       `json:"inbox_received"`
}

// ErrorRate returns the fraction of failed expeditions (0.0 to 1.0).
// Returns 0.0 when no expeditions have been recorded.
func (s *ExpeditionState) ErrorRate() float64 {
	if s.TotalExpeditions == 0 {
		return 0.0
	}
	return float64(s.Failed) / float64(s.TotalExpeditions)
}

// ProjectState replays events to produce an ExpeditionState.
// Unknown event types are silently skipped for forward compatibility.
// Returns a zero-value ExpeditionState for nil/empty input.
func ProjectState(events []domain.Event) *ExpeditionState {
	state := &ExpeditionState{}
	for _, ev := range events {
		applyEvent(state, ev)
	}
	return state
}

func applyEvent(state *ExpeditionState, ev domain.Event) {
	switch ev.Type {
	case domain.EventExpeditionCompleted:
		var data domain.ExpeditionCompletedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		state.TotalExpeditions++
		state.LastExpedition = data.Expedition
		state.LastStatus = data.Status
		state.LastExpeditionAt = ev.Timestamp
		if data.IssueID != "" {
			state.LastIssueID = data.IssueID
		}
		switch data.Status {
		case "success":
			state.Succeeded++
			state.ConsecutiveFailures = 0
		case "failed":
			state.Failed++
			state.ConsecutiveFailures++
		case "skipped":
			state.Skipped++
		}

	case domain.EventGradientChanged:
		var data domain.GradientChangedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		state.GradientLevel = data.Level

	case domain.EventDMailStaged:
		state.DMailsStaged++

	case domain.EventDMailFlushed:
		var data domain.DMailFlushedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		state.DMailsFlushed += data.Count

	case domain.EventInboxReceived:
		state.InboxReceived++

	case domain.EventGommageTriggered:
		state.GommageCount++

	case domain.EventSpecRegistered:
		// Tracked by WaveStepProgress Read Model; no counter needed in ExpeditionState.

	case domain.EventExpeditionStarted, domain.EventDMailArchived:
		// Audit-only events: no state mutation needed.

	default:
		// Unknown event type: silently skip for forward compatibility.
	}
}
