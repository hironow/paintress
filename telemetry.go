package paintress

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// tracer is a noop shim kept for root package files that still reference it.
// The real tracer lives in internal/session/telemetry.go. This shim will be
// removed when all tracer-using files have moved to the session package.
var tracer trace.Tracer = noop.NewTracerProvider().Tracer("paintress")
