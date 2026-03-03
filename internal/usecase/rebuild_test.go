package usecase

import (
	"io"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
	"github.com/hironow/paintress/internal/session"
)

func TestRebuild_InvalidCommand(t *testing.T) {
	// given: empty RepoPath
	cmd := domain.RebuildCommand{}
	logger := platform.NewLogger(io.Discard, false)

	// when
	err := Rebuild(cmd, nil, nil, logger)

	// then
	if err == nil {
		t.Fatal("expected validation error for empty RepoPath")
	}
}

func TestRebuild_EmptyEventStore(t *testing.T) {
	// given
	cmd := domain.RebuildCommand{RepoPath: "/tmp/nonexistent"}
	store := &stubEventStore{events: nil}
	applier := session.NewProjectionApplier()
	logger := platform.NewLogger(io.Discard, false)

	// when
	err := Rebuild(cmd, store, applier, logger)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applier.State().TotalExpeditions != 0 {
		t.Errorf("TotalExpeditions = %d, want 0", applier.State().TotalExpeditions)
	}
}

func TestRebuild_ReplaysEvents(t *testing.T) {
	// given
	cmd := domain.RebuildCommand{RepoPath: "/tmp/test"}
	now := time.Now()
	ev1, _ := domain.NewEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 1, Status: "success",
	}, now)
	ev2, _ := domain.NewEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 2, Status: "failed",
	}, now)
	store := &stubEventStore{events: []domain.Event{ev1, ev2}}
	applier := session.NewProjectionApplier()
	logger := platform.NewLogger(io.Discard, false)

	// when
	err := Rebuild(cmd, store, applier, logger)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := applier.State()
	if state.TotalExpeditions != 2 {
		t.Errorf("TotalExpeditions = %d, want 2", state.TotalExpeditions)
	}
	if state.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", state.Succeeded)
	}
	if state.Failed != 1 {
		t.Errorf("Failed = %d, want 1", state.Failed)
	}
}

// stubEventStore is a minimal EventStore for unit testing.
type stubEventStore struct {
	events []domain.Event
}

func (s *stubEventStore) Append(_ ...domain.Event) error       { return nil }
func (s *stubEventStore) LoadAll() ([]domain.Event, error)     { return s.events, nil }
func (s *stubEventStore) LoadSince(_ time.Time) ([]domain.Event, error) {
	return s.events, nil
}
