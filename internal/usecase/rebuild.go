package usecase

import (
	"context"
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// Rebuild replays events to regenerate projection state.
// If a SnapshotStore is provided, saves a snapshot after successful replay.
// seqAllocLatest provides the global SeqNr watermark (from SQLite counter, not event scan).
// Pass nil if SeqCounter is not available — snapshot will be saved at SeqNr=0.
func Rebuild(ctx context.Context, cmd domain.RebuildCommand, events port.EventStore, projector domain.EventApplier, snapshots port.SnapshotStore, seqAllocLatest func() (uint64, error), aggregateType string, logger domain.Logger) error {
	allEvents, loadResult, err := events.LoadAll(ctx)
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}
	if loadResult.CorruptLineCount > 0 {
		logger.Warn("event store: %d corrupt line(s) skipped", loadResult.CorruptLineCount)
	}

	logger.Info("rebuilding projections from %d event(s)", len(allEvents))

	if err := projector.Rebuild(allEvents); err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}

	// Save snapshot after successful rebuild.
	// SeqNr comes from the global counter (SQLite), NOT from scanning event files.
	// Legacy pre-cutover events may have aggregate-local SeqNr values that are
	// higher than the global counter, which would corrupt the snapshot watermark.
	if snapshots != nil {
		var latestSeqNr uint64
		if seqAllocLatest != nil {
			if seq, seqErr := seqAllocLatest(); seqErr != nil {
				logger.Warn("could not determine global SeqNr for snapshot: %v", seqErr)
			} else {
				latestSeqNr = seq
			}
		}
		state, serErr := projector.Serialize()
		if serErr != nil {
			logger.Warn("could not serialize projection for snapshot: %v", serErr)
		} else if err := snapshots.Save(ctx, aggregateType, latestSeqNr, state); err != nil {
			logger.Warn("could not save snapshot: %v", err)
		} else {
			logger.Info("snapshot saved at SeqNr=%d", latestSeqNr)
		}
	}

	logger.OK("rebuild complete")
	return nil
}
