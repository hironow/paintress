package session

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// eventStepProgressReader implements port.StepProgressReader by replaying
// events from the event store. Each call to ReadStepProgress replays all
// events to rebuild the WaveStepProgress Read Model.
type eventStepProgressReader struct {
	store port.EventStore
}

// NewStepProgressReader creates a StepProgressReader backed by the event store.
func NewStepProgressReader(store port.EventStore) port.StepProgressReader {
	return &eventStepProgressReader{store: store}
}

func (r *eventStepProgressReader) ReadStepProgress(ctx context.Context) (*domain.WaveStepProgress, error) {
	events, _, err := r.store.LoadAll()
	if err != nil {
		return nil, err
	}
	return domain.ProjectWaveStepProgress(events), nil
}
