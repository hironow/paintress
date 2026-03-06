package session_test

import (
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
	"github.com/hironow/paintress/internal/usecase/port"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// --- test helpers ---

// findSpanByName returns the first span with the given name, or nil.
func findSpanByName(spans tracetest.SpanStubs, name string) *tracetest.SpanStub {
	for i := range spans {
		if spans[i].Name == name {
			return &spans[i]
		}
	}
	return nil
}

// hasAttribute reports whether the span carries an attribute with the given key.
func hasAttribute(span *tracetest.SpanStub, key string) bool {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			return true
		}
	}
	return false
}

// --- stub EventStore ---

// stubEventStore implements port.EventStore with canned responses.
type stubEventStore struct {
	appendResult domain.AppendResult
	loadEvents   []domain.Event
	loadResult   domain.LoadResult
}

var _ port.EventStore = (*stubEventStore)(nil)

func (s *stubEventStore) Append(_ ...domain.Event) (domain.AppendResult, error) {
	return s.appendResult, nil
}

func (s *stubEventStore) LoadAll() ([]domain.Event, domain.LoadResult, error) {
	return s.loadEvents, s.loadResult, nil
}

func (s *stubEventStore) LoadSince(_ time.Time) ([]domain.Event, domain.LoadResult, error) {
	return s.loadEvents, s.loadResult, nil
}

// --- tests ---

func TestSpanEventStore_BasicMode_OmitsDebugAttributes(t *testing.T) {
	exp := setupTestTracer(t)

	prev := platform.OTELDetailLevel
	platform.OTELDetailLevel = platform.DetailBasic
	t.Cleanup(func() { platform.OTELDetailLevel = prev })

	inner := &stubEventStore{
		appendResult: domain.AppendResult{BytesWritten: 42},
		loadEvents: []domain.Event{
			{ID: "e1", Type: domain.EventDMailStaged, Timestamp: time.Now(), Data: []byte(`{"name":"x"}`)},
			{ID: "e2", Type: domain.EventDMailFlushed, Timestamp: time.Now(), Data: []byte(`{"count":1}`)},
		},
		loadResult: domain.LoadResult{FileCount: 3, CorruptLineCount: 1},
	}

	store := session.NewSpanEventStore(inner)

	// Exercise all three operations
	evt, _ := domain.NewEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "test"}, time.Now())
	store.Append(evt)  // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]
	store.LoadAll()    // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]
	store.LoadSince(time.Time{}) // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]

	spans := exp.GetSpans()

	// --- Append span ---
	appendSpan := findSpanByName(spans, "eventsource.append")
	if appendSpan == nil {
		t.Fatal("missing eventsource.append span")
	}
	if !hasAttribute(appendSpan, "event.count.in") {
		t.Error("basic mode: expected event.count.in on append span")
	}
	if hasAttribute(appendSpan, "event.append.bytes") {
		t.Error("basic mode: event.append.bytes must NOT be present on append span")
	}

	// --- LoadAll span ---
	loadAllSpan := findSpanByName(spans, "eventsource.load_all")
	if loadAllSpan == nil {
		t.Fatal("missing eventsource.load_all span")
	}
	if !hasAttribute(loadAllSpan, "event.count.out") {
		t.Error("basic mode: expected event.count.out on load_all span")
	}
	if hasAttribute(loadAllSpan, "event.file.count") {
		t.Error("basic mode: event.file.count must NOT be present on load_all span")
	}
	if hasAttribute(loadAllSpan, "event.corrupt_line.count") {
		t.Error("basic mode: event.corrupt_line.count must NOT be present on load_all span")
	}

	// --- LoadSince span ---
	loadSinceSpan := findSpanByName(spans, "eventsource.load_since")
	if loadSinceSpan == nil {
		t.Fatal("missing eventsource.load_since span")
	}
	if !hasAttribute(loadSinceSpan, "event.count.out") {
		t.Error("basic mode: expected event.count.out on load_since span")
	}
	if hasAttribute(loadSinceSpan, "event.file.count") {
		t.Error("basic mode: event.file.count must NOT be present on load_since span")
	}
	if hasAttribute(loadSinceSpan, "event.corrupt_line.count") {
		t.Error("basic mode: event.corrupt_line.count must NOT be present on load_since span")
	}
}

func TestSpanEventStore_DebugMode_IncludesAllAttributes(t *testing.T) {
	exp := setupTestTracer(t)

	prev := platform.OTELDetailLevel
	platform.OTELDetailLevel = platform.DetailDebug
	t.Cleanup(func() { platform.OTELDetailLevel = prev })

	inner := &stubEventStore{
		appendResult: domain.AppendResult{BytesWritten: 42},
		loadEvents: []domain.Event{
			{ID: "e1", Type: domain.EventDMailStaged, Timestamp: time.Now(), Data: []byte(`{"name":"x"}`)},
		},
		loadResult: domain.LoadResult{FileCount: 3, CorruptLineCount: 1},
	}

	store := session.NewSpanEventStore(inner)

	evt, _ := domain.NewEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "test"}, time.Now())
	store.Append(evt)  // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]
	store.LoadAll()    // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]
	store.LoadSince(time.Time{}) // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]

	spans := exp.GetSpans()

	// --- Append span: basic + debug attributes ---
	appendSpan := findSpanByName(spans, "eventsource.append")
	if appendSpan == nil {
		t.Fatal("missing eventsource.append span")
	}
	for _, key := range []string{"event.count.in", "event.append.bytes"} {
		if !hasAttribute(appendSpan, key) {
			t.Errorf("debug mode: expected %s on append span", key)
		}
	}

	// --- LoadAll span: basic + debug attributes ---
	loadAllSpan := findSpanByName(spans, "eventsource.load_all")
	if loadAllSpan == nil {
		t.Fatal("missing eventsource.load_all span")
	}
	for _, key := range []string{"event.count.out", "event.file.count", "event.corrupt_line.count"} {
		if !hasAttribute(loadAllSpan, key) {
			t.Errorf("debug mode: expected %s on load_all span", key)
		}
	}

	// --- LoadSince span: basic + debug attributes ---
	loadSinceSpan := findSpanByName(spans, "eventsource.load_since")
	if loadSinceSpan == nil {
		t.Fatal("missing eventsource.load_since span")
	}
	for _, key := range []string{"event.count.out", "event.file.count", "event.corrupt_line.count"} {
		if !hasAttribute(loadSinceSpan, key) {
			t.Errorf("debug mode: expected %s on load_since span", key)
		}
	}
}

func TestSpanEventStore_NoPIILeakage(t *testing.T) {
	exp := setupTestTracer(t)

	prev := platform.OTELDetailLevel
	platform.OTELDetailLevel = platform.DetailDebug // use debug for maximum attribute exposure
	t.Cleanup(func() { platform.OTELDetailLevel = prev })

	inner := &stubEventStore{
		appendResult: domain.AppendResult{BytesWritten: 128},
		loadEvents: []domain.Event{
			{ID: "e1", Type: domain.EventDMailStaged, Timestamp: time.Now(), Data: []byte(`{"name":"secret-dmail"}`)},
		},
		loadResult: domain.LoadResult{FileCount: 1, CorruptLineCount: 0},
	}

	store := session.NewSpanEventStore(inner)

	evt, _ := domain.NewEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "secret-dmail"}, time.Now())
	store.Append(evt)  // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]
	store.LoadAll()    // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]
	store.LoadSince(time.Time{}) // nosemgrep: adr0003-otel-span-without-defer-end -- test exercises wrapper [permanent]

	spans := exp.GetSpans()

	// Prohibited attribute patterns: event bodies, event IDs, raw JSON data
	prohibited := []string{
		"event.body",
		"event.data",
		"event.payload",
		"event.id",
		"event.ids",
		"event.content",
	}
	for _, s := range spans {
		for _, key := range prohibited {
			if hasAttribute(&s, key) {
				t.Errorf("PII leak: span %q must NOT contain attribute %q", s.Name, key)
			}
		}
		// Also verify no attribute value contains the raw event body
		for _, attr := range s.Attributes {
			v := attr.Value.AsString()
			if v == "" {
				continue
			}
			if strings.Contains(v, "secret-dmail") {
				t.Errorf("PII leak: span %q attribute %q contains event body data %q", s.Name, string(attr.Key), v)
			}
		}
	}
}
