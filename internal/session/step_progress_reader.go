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
	store  port.EventStore
	logger domain.Logger
}

// NewStepProgressReader creates a StepProgressReader backed by the event store.
func NewStepProgressReader(store port.EventStore, logger domain.Logger) port.StepProgressReader {
	return &eventStepProgressReader{store: store, logger: logger}
}

func (r *eventStepProgressReader) ReadStepProgress(ctx context.Context) (*domain.WaveStepProgress, error) {
	events, loadResult, err := r.store.LoadAll(ctx)
	if err != nil {
		return nil, err
	}
	if loadResult.CorruptLineCount > 0 && r.logger != nil {
		r.logger.Warn("step_progress_reader: %d corrupt event lines skipped", loadResult.CorruptLineCount)
	}
	return domain.ProjectWaveStepProgress(events), nil
}
