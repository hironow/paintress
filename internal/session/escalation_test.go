package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

func TestStageEscalation_StagesFeedbackDMail(t *testing.T) {
	// given
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	p := &Paintress{
		config:      domain.Config{Continent: continent},
		outboxStore: store,
		Logger:      platform.NewLogger(nil, false),
	}

	// when — stageEscalation calls SendDMail which does Stage + Flush internally
	p.stageEscalation(context.Background(), 5, 3)

	// then — verify file exists in outbox (SendDMail already flushed)
	outboxDir := domain.OutboxDir(continent)
	entries, err := os.ReadDir(outboxDir)
	if err != nil {
		t.Fatalf("ReadDir outbox: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("outbox file count = %d, want 1", len(entries))
	}

	// verify content is feedback kind
	data, err := os.ReadFile(filepath.Join(outboxDir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	dm, err := domain.ParseDMail(data)
	if err != nil {
		t.Fatalf("ParseDMail: %v", err)
	}
	if dm.Kind != "feedback" {
		t.Errorf("Kind = %q, want %q", dm.Kind, "feedback")
	}
	if dm.Severity != "high" {
		t.Errorf("Severity = %q, want %q", dm.Severity, "high")
	}
}

func TestStageEscalation_ArchiveAndOutbox(t *testing.T) {
	// given
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	p := &Paintress{
		config:      domain.Config{Continent: continent},
		outboxStore: store,
		Logger:      platform.NewLogger(nil, false),
	}

	// when
	p.stageEscalation(context.Background(), 5, 3)

	// then — both archive and outbox should have the file
	archiveDir := domain.ArchiveDir(continent)
	outboxDir := domain.OutboxDir(continent)

	archiveEntries, _ := os.ReadDir(archiveDir)
	outboxEntries, _ := os.ReadDir(outboxDir)

	if len(archiveEntries) != 1 {
		t.Errorf("archive file count = %d, want 1", len(archiveEntries))
	}
	if len(outboxEntries) != 1 {
		t.Errorf("outbox file count = %d, want 1", len(outboxEntries))
	}
}

func TestStageEscalation_Idempotent(t *testing.T) {
	// given
	continent := t.TempDir()
	ensureExpeditionDirs(t, continent)
	store := testOutboxStore(t, continent)
	p := &Paintress{
		config:      domain.Config{Continent: continent},
		outboxStore: store,
		Logger:      platform.NewLogger(nil, false),
	}

	// when — stage twice with same expedition (same D-Mail name)
	p.stageEscalation(context.Background(), 5, 3)
	p.stageEscalation(context.Background(), 5, 3)

	// then — only one D-Mail in outbox (INSERT OR IGNORE + already flushed)
	outboxDir := domain.OutboxDir(continent)
	entries, _ := os.ReadDir(outboxDir)
	if len(entries) != 1 {
		t.Errorf("outbox file count = %d, want 1 (idempotent)", len(entries))
	}
}

func TestStageEscalation_NilOutboxStore(t *testing.T) {
	// given — no outbox store configured
	p := &Paintress{
		config: domain.Config{Continent: t.TempDir()},
		Logger: platform.NewLogger(nil, false),
	}

	// when / then — should not panic
	p.stageEscalation(context.Background(), 5, 3)
}

func TestHandleEscalation_ReturnsErrorOnEmitterFail(t *testing.T) {
	// given — emitter that always fails
	emitter := &failingEmitter{err: fmt.Errorf("disk full")}
	p := &Paintress{
		config:  domain.Config{Continent: t.TempDir()},
		Logger:  platform.NewLogger(nil, false),
		Emitter: emitter,
	}
	dm := domain.DMail{Name: "esc-fail", Issues: []string{"MY-99"}}

	// when
	err := p.handleEscalation(dm)

	// then — error must be propagated (escalation events are critical)
	if err == nil {
		t.Fatal("expected error from failing emitter, got nil")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("error should contain root cause, got: %s", err.Error())
	}
}

func TestHandleEscalation_SucceedsWithWorkingEmitter(t *testing.T) {
	// given — emitter that succeeds
	emitter := &nopEmitter{}
	p := &Paintress{
		config:  domain.Config{Continent: t.TempDir()},
		Logger:  platform.NewLogger(nil, false),
		Emitter: emitter,
	}
	dm := domain.DMail{Name: "esc-ok", Issues: []string{"MY-100"}}

	// when
	err := p.handleEscalation(dm)

	// then — no error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// nopEmitter is a no-op ExpeditionEventEmitter that always succeeds.
type nopEmitter struct{}

func (n *nopEmitter) EmitStartExpedition(_, _ int, _ string, _ time.Time) error    { return nil }
func (n *nopEmitter) EmitCompleteExpedition(_ int, _, _, _ string, _ time.Time) error {
	return nil
}
func (n *nopEmitter) EmitInboxReceived(_, _ string, _ time.Time) error             { return nil }
func (n *nopEmitter) EmitGommage(_ int, _ time.Time) error                         { return nil }
func (n *nopEmitter) EmitGradientChange(_ int, _ string, _ time.Time) error        { return nil }
func (n *nopEmitter) EmitRetryAttempted(_ string, _ int, _ time.Time) error        { return nil }
func (n *nopEmitter) EmitEscalated(_ string, _ []string, _ time.Time) error        { return nil }
func (n *nopEmitter) EmitDMailStaged(_ string, _ time.Time) error                  { return nil }
func (n *nopEmitter) EmitDMailFlushed(_ int, _ time.Time) error                    { return nil }
func (n *nopEmitter) EmitDMailArchived(_ string, _ time.Time) error                { return nil }
