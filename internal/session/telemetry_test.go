package session_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// setupTestTracer installs an InMemoryExporter with a synchronous span processor
// so spans are immediately available for inspection. It restores the global
// TracerProvider after the test.
func setupTestTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	platform.Tracer = tp.Tracer("paintress-test")
	t.Cleanup(func() {
		tp.Shutdown(context.Background())
		otel.SetTracerProvider(prev)
		// Restore noop tracer so other tests are not affected
		platform.Tracer = prev.Tracer("paintress")
	})
	return exp
}

func TestInitTracer_ShutdownFlushesSpans(t *testing.T) {
	exp := setupTestTracer(t)

	_, span := platform.Tracer.Start(context.Background(), "flushed-span") // nosemgrep: adr0003-otel-span-without-defer-end -- test span, immediately ended [permanent]
	span.End()

	spans := exp.GetSpans()
	if len(spans) == 0 {
		t.Fatal("expected at least 1 span in InMemoryExporter after span.End()")
	}
	if spans[0].Name != "flushed-span" {
		t.Errorf("span name = %q, want %q", spans[0].Name, "flushed-span")
	}
}

func TestSpan_PaintressRun_CreatesRootSpan(t *testing.T) {
	exp := setupTestTracer(t)

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	cfg := domain.Config{
		Continent:      dir,
		MaxExpeditions: 1,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DryRun:         true,
	}

	p := session.NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.Run(context.Background())

	spans := exp.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "paintress.run" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, len(spans))
		for i, s := range spans {
			names[i] = s.Name
		}
		t.Errorf("expected 'paintress.run' span, got spans: %v", names)
	}
}

func TestSpan_Expedition_HasAttributes(t *testing.T) {
	exp := setupTestTracer(t)

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)

	cfg := domain.Config{
		Continent:      dir,
		MaxExpeditions: 1,
		TimeoutSec:     30,
		Model:          "opus",
		BaseBranch:     "main",
		DryRun:         true,
	}

	p := session.NewPaintress(cfg, platform.NewLogger(io.Discard, false), io.Discard, io.Discard, nil, nil, nil, nil)
	p.Run(context.Background())

	spans := exp.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "expedition" {
			found = true
			for _, attr := range s.Attributes {
				if string(attr.Key) == "expedition.number" {
					if attr.Value.AsInt64() < 1 {
						t.Errorf("expedition.number = %d, want >= 1", attr.Value.AsInt64())
					}
				}
			}
			break
		}
	}
	if !found {
		names := make([]string, len(spans))
		for i, s := range spans {
			names[i] = s.Name
		}
		t.Errorf("expected 'expedition' span, got spans: %v", names)
	}
}

func TestSpan_ClaudeInvoke_RecordsTimeoutEvent(t *testing.T) {
	exporter := setupTestTracer(t)

	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition", "journal"), 0755)
	os.MkdirAll(filepath.Join(dir, ".expedition", ".run", "logs"), 0755)

	// Script that sleeps longer than the timeout
	sleepScript := filepath.Join(dir, "slowclaude.sh")
	os.WriteFile(sleepScript, []byte("#!/bin/bash\nexec sleep 999\n"), 0755)

	// Minimal HTTP server for DevServer
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot bind port: %v", err)
	}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Listener = ln
	srv.Start()
	defer srv.Close()

	e := &session.Expedition{
		Number:    1,
		Continent: dir,
		Config: domain.Config{
			Continent:  dir,
			TimeoutSec: 1, // 1 second timeout
			ClaudeCmd:  sleepScript,
			BaseBranch: "main",
			Model:      "opus",
		},
		LogDir:   filepath.Join(dir, ".expedition", ".run", "logs"),
		Logger:   platform.NewLogger(io.Discard, false),
		Gradient: domain.NewGradientGauge(5),
		Reserve:  domain.NewReserveParty("opus", nil, platform.NewLogger(io.Discard, false)),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	e.Run(ctx)

	spans := exporter.GetSpans()
	var found bool
	for _, s := range spans {
		if s.Name == "claude.invoke" {
			for _, ev := range s.Events {
				if ev.Name == "expedition.timeout" {
					found = true
				}
			}

			// Verify gen_ai.* semantic convention attributes (P1-3)
			requiredAttrs := map[string]string{
				"gen_ai.operation.name": "chat",
				"gen_ai.system":         "anthropic",
			}
			for key, want := range requiredAttrs {
				var attrFound bool
				for _, attr := range s.Attributes {
					if string(attr.Key) == key {
						attrFound = true
						if got := attr.Value.AsString(); got != want {
							t.Errorf("attr %s = %q, want %q", key, got, want)
						}
					}
				}
				if !attrFound {
					t.Errorf("missing gen_ai attribute %q on claude.invoke span", key)
				}
			}

			// gen_ai.request.model should be present
			var modelFound bool
			for _, attr := range s.Attributes {
				if string(attr.Key) == "gen_ai.request.model" {
					modelFound = true
				}
			}
			if !modelFound {
				t.Error("missing gen_ai.request.model attribute on claude.invoke span")
			}

			// Cross-tool conformance: claude.model and claude.timeout_sec must be present
			conformanceAttrs := []string{"claude.model", "claude.timeout_sec"}
			for _, key := range conformanceAttrs {
				var attrFound bool
				for _, attr := range s.Attributes {
					if string(attr.Key) == key {
						attrFound = true
					}
				}
				if !attrFound {
					t.Errorf("missing cross-tool conformance attribute %q on claude.invoke span", key)
				}
			}
		}
	}
	if !found {
		t.Error("expected 'expedition.timeout' event on 'claude.invoke' span")
	}
}
