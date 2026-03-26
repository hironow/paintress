package domain

// white-box-reason: tests unexported seqNr field increment on ExpeditionAggregate

import (
	"testing"
	"time"
)

func TestExpeditionAggregate_SeqNrIncrements(t *testing.T) {
	agg := NewExpeditionAggregate()
	now := time.Now()

	ev1, err := agg.StartExpedition(1, 0, "opus", now)
	if err != nil {
		t.Fatal(err)
	}

	if ev1.SeqNr != 1 {
		t.Errorf("ev1.SeqNr = %d, want 1", ev1.SeqNr)
	}
	if ev1.AggregateType != AggregateTypeExpedition {
		t.Errorf("ev1.AggregateType = %q, want %q", ev1.AggregateType, AggregateTypeExpedition)
	}

	// Second event should increment
	ev2, err := agg.RecordGradientChange(3, "charge", now)
	if err != nil {
		t.Fatal(err)
	}
	if ev2.SeqNr != 2 {
		t.Errorf("ev2.SeqNr = %d, want 2", ev2.SeqNr)
	}
}
