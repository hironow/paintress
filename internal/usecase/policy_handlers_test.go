package usecase

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

func TestPolicyHandler_ExpeditionCompleted_InfoOutput(t *testing.T) {
	// given
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false)
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger)

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

func TestPolicyHandler_ExpeditionCompleted_DebugOnly_NoInfoOutput(t *testing.T) {
	// given: Debug-only handler (gradient.changed) should NOT produce Info output
	var buf bytes.Buffer
	logger := platform.NewLogger(&buf, false) // verbose=false so Debug is suppressed
	engine := NewPolicyEngine(logger)
	registerExpeditionPolicies(engine, logger)

	ev, err := domain.NewEvent(domain.EventGradientChanged, domain.GradientChangedData{
		Level: 3, Operator: "auto",
	}, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	// when
	engine.Dispatch(context.Background(), ev)

	// then: no output (Debug suppressed when verbose=false)
	output := buf.String()
	if output != "" {
		t.Errorf("expected no output for Debug-only handler with verbose=false, got: %s", output)
	}
}
