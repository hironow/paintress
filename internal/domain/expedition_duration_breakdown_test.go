package domain_test

import (
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

func TestExpeditionDurationBreakdown_Fields(t *testing.T) {
	// given
	promptDur := 150 * time.Millisecond
	invokeDur := 30 * time.Second

	// when
	bd := domain.ExpeditionDurationBreakdown{
		PromptBuildDuration: promptDur,
		InvokeDuration:      invokeDur,
	}

	// then
	if bd.PromptBuildDuration != promptDur {
		t.Errorf("PromptBuildDuration = %v, want %v", bd.PromptBuildDuration, promptDur)
	}
	if bd.InvokeDuration != invokeDur {
		t.Errorf("InvokeDuration = %v, want %v", bd.InvokeDuration, invokeDur)
	}
}

func TestExpeditionDurationBreakdown_Total(t *testing.T) {
	// given
	bd := domain.ExpeditionDurationBreakdown{
		PromptBuildDuration: 200 * time.Millisecond,
		InvokeDuration:      10 * time.Second,
	}

	// when
	total := bd.Total()

	// then
	want := 200*time.Millisecond + 10*time.Second
	if total != want {
		t.Errorf("Total() = %v, want %v", total, want)
	}
}

func TestExpeditionDurationBreakdown_SpanAttributes(t *testing.T) {
	// given
	bd := domain.ExpeditionDurationBreakdown{
		PromptBuildDuration: 300 * time.Millisecond,
		InvokeDuration:      45 * time.Second,
	}

	// when
	attrs := bd.SpanAttributes()

	// then: must contain prompt_build_ms and invoke_ms
	attrMap := make(map[string]any)
	for _, a := range attrs {
		attrMap[string(a.Key)] = a.Value.AsInt64()
	}

	if _, ok := attrMap["expedition.prompt_build_ms"]; !ok {
		t.Error("SpanAttributes() must include expedition.prompt_build_ms")
	}
	if _, ok := attrMap["expedition.invoke_ms"]; !ok {
		t.Error("SpanAttributes() must include expedition.invoke_ms")
	}
	if got := attrMap["expedition.prompt_build_ms"]; got != int64(300) {
		t.Errorf("expedition.prompt_build_ms = %v, want 300", got)
	}
	if got := attrMap["expedition.invoke_ms"]; got != int64(45000) {
		t.Errorf("expedition.invoke_ms = %v, want 45000", got)
	}
}
