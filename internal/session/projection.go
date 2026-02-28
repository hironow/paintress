package session

import (
	"encoding/json"

	"github.com/hironow/paintress"
)

// ExpeditionState is the materialized READ MODEL projected from events.
type ExpeditionState struct {
	TotalExpeditions int
	Succeeded        int
	Failed           int
	Skipped          int
	LastExpedition   int
	LastStatus       string
	GradientLevel    int
	DMailsStaged     int
	DMailsFlushed    int
	InboxReceived    int
}

// ProjectState replays events to produce an ExpeditionState.
// Unknown event types are silently skipped for forward compatibility.
// Returns a zero-value ExpeditionState for nil/empty input.
func ProjectState(events []paintress.Event) *ExpeditionState {
	state := &ExpeditionState{}
	for _, ev := range events {
		applyEvent(state, ev)
	}
	return state
}

func applyEvent(state *ExpeditionState, ev paintress.Event) {
	switch ev.Type {
	case paintress.EventExpeditionCompleted:
		var data paintress.ExpeditionCompletedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		state.TotalExpeditions++
		state.LastExpedition = data.Expedition
		state.LastStatus = data.Status
		switch data.Status {
		case "success":
			state.Succeeded++
		case "failed":
			state.Failed++
		case "skipped":
			state.Skipped++
		}

	case paintress.EventGradientChanged:
		var data paintress.GradientChangedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		state.GradientLevel = data.Level

	case paintress.EventDMailStaged:
		state.DMailsStaged++

	case paintress.EventDMailFlushed:
		var data paintress.DMailFlushedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		state.DMailsFlushed += data.Count

	case paintress.EventInboxReceived:
		state.InboxReceived++

	case paintress.EventExpeditionStarted, paintress.EventDMailArchived, paintress.EventGommageTriggered:
		// Audit-only events: no state mutation needed.

	default:
		// Unknown event type: silently skip for forward compatibility.
	}
}
