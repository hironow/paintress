package usecase

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/usecase/port"
)

type notifyCall struct {
	title   string
	message string
}

type spyNotifier struct {
	calls []notifyCall
}

type metricsCall struct {
	eventType string
	status    string
}

type spyPolicyMetrics struct {
	calls []metricsCall
}

func (s *spyPolicyMetrics) RecordPolicyEvent(_ context.Context, eventType, status string) {
	s.calls = append(s.calls, metricsCall{eventType: eventType, status: status})
}

func (s *spyNotifier) Notify(_ context.Context, title, message string) error {
	s.calls = append(s.calls, notifyCall{title: title, message: message})
	return nil
}

func TestPolicyHandler_ExpeditionCompleted_InfoOutput(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, &port.NopNotifier{}, &port.NopPolicyMetrics{})

	ev, err := domain.NewEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 42, Status: "success",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then: Info-level output should contain expedition number and status
	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("expected INFO level output, got: %s", output)
	}
	if !strings.Contains(output, "#42") {
		t.Errorf("expected expedition number in output, got: %s", output)
	}
	if !strings.Contains(output, "success") {
		t.Errorf("expected status in output, got: %s", output)
	}
}

func TestPolicyHandler_ExpeditionCompleted_NotifiesSideEffect(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	spy := &spyNotifier{}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, spy, &port.NopPolicyMetrics{})

	ev, err := domain.NewEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 42, Status: "success",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then: Notify should have been called once
	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 Notify call, got %d", len(spy.calls))
	}
	call := spy.calls[0]
	if !strings.Contains(call.title, "Paintress") {
		t.Errorf("expected title to contain 'Paintress', got: %s", call.title)
	}
	if !strings.Contains(call.message, "#42") {
		t.Errorf("expected message to contain expedition number, got: %s", call.message)
	}
	if !strings.Contains(call.message, "success") {
		t.Errorf("expected message to contain status, got: %s", call.message)
	}
}

func TestPolicyHandler_ExpeditionCompleted_RecordsMetrics(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	spy := &spyPolicyMetrics{}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, &port.NopNotifier{}, spy)

	ev, err := domain.NewEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 42, Status: "success",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then
	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 RecordPolicyEvent call, got %d", len(spy.calls))
	}
	if spy.calls[0].eventType != "expedition.completed" {
		t.Errorf("expected eventType 'expedition.completed', got: %s", spy.calls[0].eventType)
	}
	if spy.calls[0].status != "handled" {
		t.Errorf("expected status 'handled', got: %s", spy.calls[0].status)
	}
}

func TestPolicyHandler_InboxReceived_RecordsMetrics(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	spy := &spyPolicyMetrics{}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, &port.NopNotifier{}, spy)

	ev, err := domain.NewEvent(domain.EventInboxReceived, map[string]string{
		"source": "test",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then
	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 RecordPolicyEvent call, got %d", len(spy.calls))
	}
	if spy.calls[0].eventType != "inbox.received" {
		t.Errorf("expected eventType 'inbox.received', got: %s", spy.calls[0].eventType)
	}
}

func TestPolicyHandler_GradientChanged_RecordsMetrics(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	spy := &spyPolicyMetrics{}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, &port.NopNotifier{}, spy)

	ev, err := domain.NewEvent(domain.EventGradientChanged, domain.GradientChangedData{
		Level: 3, Operator: "auto",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then
	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 RecordPolicyEvent call, got %d", len(spy.calls))
	}
	if spy.calls[0].eventType != "gradient.changed" {
		t.Errorf("expected eventType 'gradient.changed', got: %s", spy.calls[0].eventType)
	}
}

func TestPolicyHandler_DMailStaged_RecordsMetrics(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	spy := &spyPolicyMetrics{}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, &port.NopNotifier{}, spy)

	ev, err := domain.NewEvent(domain.EventDMailStaged, map[string]string{
		"kind": "feedback",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then
	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 RecordPolicyEvent call, got %d", len(spy.calls))
	}
	if spy.calls[0].eventType != "dmail.staged" {
		t.Errorf("expected eventType 'dmail.staged', got: %s", spy.calls[0].eventType)
	}
}

func TestPolicyHandler_GradientChanged_NotifiesSideEffect(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	spy := &spyNotifier{}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, spy, &port.NopPolicyMetrics{})

	ev, err := domain.NewEvent(domain.EventGradientChanged, domain.GradientChangedData{
		Level: 3, Operator: "charge",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then
	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 Notify call, got %d", len(spy.calls))
	}
	call := spy.calls[0]
	if !strings.Contains(call.title, "Paintress") {
		t.Errorf("expected title to contain 'Paintress', got: %s", call.title)
	}
	if !strings.Contains(call.message, "Gradient") {
		t.Errorf("expected message to contain 'Gradient', got: %s", call.message)
	}
}

func TestPolicyHandler_DMailStaged_NotifiesSideEffect(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	spy := &spyNotifier{}
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger, spy, &port.NopPolicyMetrics{})

	ev, err := domain.NewEvent(domain.EventDMailStaged, map[string]string{
		"kind": "feedback",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then
	if len(spy.calls) != 1 {
		t.Fatalf("expected 1 Notify call, got %d", len(spy.calls))
	}
	call := spy.calls[0]
	if !strings.Contains(call.title, "Paintress") {
		t.Errorf("expected title to contain 'Paintress', got: %s", call.title)
	}
	if !strings.Contains(call.message, "D-Mail staged") {
		t.Errorf("expected message to contain 'D-Mail staged', got: %s", call.message)
	}
}
