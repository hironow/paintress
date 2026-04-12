package session_test

// white-box-reason: verifies telemetry span attributes from Expedition.Run

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
)

func TestExpedition_Run_RecordsPromptBuildDurationOnSpan(t *testing.T) {
	// given: a test tracer
	exporter := setupTestTracer(t)

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run", "logs"), 0755)

	// Script that exits quickly
	sleepScript := filepath.Join(dir, "fastclaude.sh")
	os.WriteFile(sleepScript, []byte("#!/bin/bash\nexit 0\n"), 0755)

	e := &session.Expedition{
		Number:    1,
		Continent: dir,
		Config: domain.Config{
			Continent:  dir,
			TimeoutSec: 5,
			ClaudeCmd:  sleepScript,
			BaseBranch: "main",
			Model:      "opus",
		},
		LogDir:   filepath.Join(dir, ".expedition", ".run", "logs"),
		Logger:   platform.NewLogger(io.Discard, false),
		Gradient: harness.NewGradientGauge(5),
		Reserve:  harness.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// when
	e.Run(ctx)

	// then: the provider.invoke span must have expedition.prompt_build_ms attribute
	spans := exporter.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name != "provider.invoke" {
			continue
		}
		for _, attr := range s.Attributes {
			if string(attr.Key) == "expedition.prompt_build_ms" {
				found = true
				if attr.Value.AsInt64() < 0 {
					t.Errorf("expedition.prompt_build_ms = %d, want >= 0", attr.Value.AsInt64())
				}
			}
		}
	}
	if !found {
		names := make([]string, 0, len(spans))
		for _, s := range spans {
			names = append(names, s.Name)
		}
		t.Errorf("expected expedition.prompt_build_ms attribute on provider.invoke span, spans seen: %v", names)
	}
}
