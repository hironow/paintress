package domain

import "encoding/json"

// WaveStepProgress is the CQRS Read Model for wave step completion tracking.
// Built by replaying events from the event store (spec.registered + expedition.completed).
// Pure data structure: no I/O, no context.Context.
type WaveStepProgress struct {
	waves      map[string]*waveStepEntry // waveID → entry
	waveOrder  []string                  // preserve registration order
	stepTitles map[string]string         // "waveID:stepID" → title
}

type waveStepEntry struct {
	steps map[string]StepStatus // stepID → status
	order []string              // preserve step order
}

// ProjectWaveStepProgress replays events to build the WaveStepProgress Read Model.
// Pure function: same pattern as ProjectState().
//
// Recognized events:
//   - spec.registered: registers wave and its steps as pending (first-wins for idempotency)
//   - expedition.completed: marks step as completed (success/skipped) or retains as pending (failed)
func ProjectWaveStepProgress(events []Event) *WaveStepProgress {
	state := &WaveStepProgress{
		waves:      make(map[string]*waveStepEntry),
		stepTitles: make(map[string]string),
	}
	for _, ev := range events {
		applyWaveStepEvent(state, ev)
	}
	return state
}

func applyWaveStepEvent(state *WaveStepProgress, ev Event) {
	switch ev.Type {
	case EventSpecRegistered:
		var data SpecRegisteredData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		if _, exists := state.waves[data.WaveID]; exists {
			return // first spec wins — immutable
		}
		entry := &waveStepEntry{
			steps: make(map[string]StepStatus),
		}
		if len(data.Steps) == 0 {
			// Single-step wave: use waveID as implicit stepID
			entry.steps[data.WaveID] = StepPending
			entry.order = []string{data.WaveID}
			state.stepTitles[data.WaveID+":"+data.WaveID] = data.WaveID
		} else {
			for _, s := range data.Steps {
				entry.steps[s.ID] = StepPending
				entry.order = append(entry.order, s.ID)
				state.stepTitles[data.WaveID+":"+s.ID] = s.Title
			}
		}
		state.waves[data.WaveID] = entry
		state.waveOrder = append(state.waveOrder, data.WaveID)

	case EventExpeditionCompleted:
		var data ExpeditionCompletedData
		if err := json.Unmarshal(ev.Data, &data); err != nil {
			return
		}
		if data.WaveID == "" {
			return // non-wave expedition or legacy event
		}
		entry, ok := state.waves[data.WaveID]
		if !ok {
			return // completed before spec registered — ignore
		}
		stepID := data.StepID
		if stepID == "" {
			stepID = data.WaveID // single-step wave
		}
		if _, exists := entry.steps[stepID]; !exists {
			return // unknown step
		}
		switch data.Status {
		case "success", "skipped":
			entry.steps[stepID] = StepCompleted
		case "failed", "parse_error":
			// failed remains pending for retry — no state change
		}
	}
}

// HasWaves returns true when at least one spec.registered event has been processed.
func (w *WaveStepProgress) HasWaves() bool {
	return len(w.waves) > 0
}

// PendingTargets returns ExpeditionTargets for steps not yet completed.
// Preserves wave registration order and step definition order.
func (w *WaveStepProgress) PendingTargets() []ExpeditionTarget {
	var targets []ExpeditionTarget
	for _, waveID := range w.waveOrder {
		entry := w.waves[waveID]
		for _, stepID := range entry.order {
			if entry.steps[stepID] != StepPending {
				continue
			}
			id := waveID + ":" + stepID
			if stepID == waveID {
				id = waveID // single-step wave
			}
			targets = append(targets, ExpeditionTarget{
				ID:     id,
				WaveID: waveID,
				StepID: stepID,
				Title:  w.stepTitles[waveID+":"+stepID],
			})
		}
	}
	return targets
}
