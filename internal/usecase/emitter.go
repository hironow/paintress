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
	agg        *domain.ExpeditionAggregate
	store      port.EventStore
	dispatcher port.EventDispatcher
	logger     domain.Logger
}

// NewExpeditionEventEmitter creates an ExpeditionEventEmitter that wraps the aggregate event chain.
func NewExpeditionEventEmitter(
	agg *domain.ExpeditionAggregate,
	store port.EventStore,
	dispatcher port.EventDispatcher,
	logger domain.Logger,
) port.ExpeditionEventEmitter {
	return &expeditionEventEmitter{
		agg:        agg,
		store:      store,
		dispatcher: dispatcher,
		logger:     logger,
	}
}

// emit persists events and dispatches (best-effort).
func (e *expeditionEventEmitter) emit(events ...domain.Event) error {
	if e.store != nil {
		if _, err := e.store.Append(events...); err != nil {
			return fmt.Errorf("append events: %w", err)
		}
	}
	if e.dispatcher != nil {
		for _, ev := range events {
			if err := e.dispatcher.Dispatch(context.Background(), ev); err != nil {
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

func (e *expeditionEventEmitter) EmitCompleteExpedition(expedition int, status, issueID, bugsFound string, now time.Time) error {
	events, err := e.agg.CompleteExpedition(expedition, status, issueID, bugsFound, now)
	if err != nil {
		return err
	}
	return e.emit(events...)
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
