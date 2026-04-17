package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/usecase/port"
)

// expeditionEventEmitter implements port.ExpeditionEventEmitter.
// It wraps the aggregate + event store + dispatcher.
// Emit chain: agg.Record*() → store.Append() → dispatch (best-effort).
type expeditionEventEmitter struct {
	agg          *domain.ExpeditionAggregate
	store        port.EventStore
	dispatcher   port.EventDispatcher
	logger       domain.Logger
	seqAlloc     port.SeqAllocator
	expeditionID string // enriches events with correlation metadata
	prevID       string // previous event ID for causation chain
	ctx          context.Context //nolint:containedctx // stored for trace propagation into emit chain
}

// SetSeqAllocator injects a SeqAllocator for SeqNr allocation into emitted events.
func (e *expeditionEventEmitter) SetSeqAllocator(alloc port.SeqAllocator) {
	e.seqAlloc = alloc
}

// NewExpeditionEventEmitter creates an ExpeditionEventEmitter that wraps the aggregate event chain.
// ctx is used for trace propagation in store/dispatcher operations.
func NewExpeditionEventEmitter(
	ctx context.Context,
	agg *domain.ExpeditionAggregate,
	store port.EventStore,
	dispatcher port.EventDispatcher,
	logger domain.Logger,
	expeditionID string,
) port.ExpeditionEventEmitter {
	return &expeditionEventEmitter{
		agg:          agg,
		store:        store,
		dispatcher:   dispatcher,
		logger:       logger,
		expeditionID: expeditionID,
		ctx:          ctx,
	}
}

// emit enriches events with correlation metadata, persists, and dispatches.
func (e *expeditionEventEmitter) emit(events ...domain.Event) error {
	ctx := e.ctx
	for i := range events {
		events[i].CorrelationID = e.expeditionID
		if e.prevID != "" {
			events[i].CausationID = e.prevID
		}
		if e.seqAlloc != nil {
			seq, err := e.seqAlloc.AllocSeqNr(ctx)
			if err != nil {
				return fmt.Errorf("alloc seq nr: %w", err)
			}
			events[i].SeqNr = seq
		}
	}
	if e.store != nil {
		if _, err := e.store.Append(ctx, events...); err != nil {
			return fmt.Errorf("append events: %w", err)
		}
	}
	// Update causation chain after successful store
	if len(events) > 0 {
		e.prevID = events[len(events)-1].ID
	}
	if e.dispatcher != nil {
		for _, ev := range events {
			if err := e.dispatcher.Dispatch(ctx, ev); err != nil {
				e.logger.Warn("policy dispatch %s: %v", ev.Type, err)
			}
		}
	}
	return nil
}

func (e *expeditionEventEmitter) EmitStartExpedition(expedition, worker int, model string, now time.Time) error {
	ev, err := e.agg.StartExpedition(expedition, worker, model, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitCompleteExpedition(expedition int, status, issueID, bugsFound, waveID, stepID string, now time.Time) error { // nosemgrep: domain-primitives.multiple-string-params-go -- status/issueID/bugsFound/waveID/stepID are semantically distinct [permanent]
	events, err := e.agg.CompleteExpedition(expedition, status, issueID, bugsFound, waveID, stepID, now)
	if err != nil {
		return err
	}
	return e.emit(events...)
}

func (e *expeditionEventEmitter) EmitSpecRegistered(waveID string, steps []domain.WaveStepDef, source string, now time.Time) error {
	ev, err := e.agg.RecordSpecRegistered(waveID, steps, source, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitInboxReceived(name, severity string, now time.Time) error {
	ev, err := e.agg.RecordInboxReceived(name, severity, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitGommage(expedition int, now time.Time) error {
	ev, err := e.agg.RecordGommage(expedition, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitGradientChange(level int, operator string, now time.Time) error {
	ev, err := e.agg.RecordGradientChange(level, operator, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitRetryAttempted(dmailKey string, attempt int, now time.Time) error {
	ev, err := e.agg.RecordRetryAttempted(dmailKey, attempt, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitEscalated(dmailName string, issues []string, now time.Time) error {
	ev, err := e.agg.RecordEscalated(dmailName, issues, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitResolved(dmailName string, issues []string, now time.Time) error {
	ev, err := e.agg.RecordResolved(dmailName, issues, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitDMailStaged(name string, now time.Time) error {
	ev, err := e.agg.RecordDMailStaged(name, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitDMailFlushed(count int, now time.Time) error {
	ev, err := e.agg.RecordDMailFlushed(count, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitDMailArchived(name string, now time.Time) error {
	ev, err := e.agg.RecordDMailArchived(name, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitGommageRecovery(expedition int, class, action string, retryNum int, cooldown string, now time.Time) error {
	ev, err := e.agg.RecordGommageRecovery(expedition, domain.GommageClass(class), action, retryNum, cooldown, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}

func (e *expeditionEventEmitter) EmitCheckpoint(expedition int, phase, workDir string, commitCount int, now time.Time) error {
	ev, err := e.agg.RecordCheckpoint(expedition, phase, workDir, commitCount, now)
	if err != nil {
		return err
	}
	return e.emit(ev)
}
