package usecase

import (
	"context"
	"fmt"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// Rebuild replays events to regenerate projection state.
// If a SnapshotStore is provided, saves a snapshot after successful replay.
func Rebuild(cmd domain.RebuildCommand, events port.EventStore, projector domain.EventApplier, snapshots port.SnapshotStore, aggregateType string, logger domain.Logger) error {
	allEvents, _, err := events.LoadAll()
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}

	logger.Info("rebuilding projections from %d event(s)", len(allEvents))

	if err := projector.Rebuild(allEvents); err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}

	// Save snapshot after successful rebuild
	if snapshots != nil {
		latestSeqNr, seqErr := events.LatestSeqNr()
		if seqErr != nil {
			logger.Warn("could not determine latest SeqNr for snapshot: %v", seqErr)
		} else {
			state, serErr := projector.Serialize()
			if serErr != nil {
				logger.Warn("could not serialize projection for snapshot: %v", serErr)
			} else if err := snapshots.Save(context.Background(), aggregateType, latestSeqNr, state); err != nil {
				logger.Warn("could not save snapshot: %v", err)
			} else {
				logger.Info("snapshot saved at SeqNr=%d", latestSeqNr)
			}
		}
	}

	logger.OK("rebuild complete")
	return nil
}
