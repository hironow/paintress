package session_test

import (
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestProjectionApplier_Apply_UpdatesState(t *testing.T) {
	// given
	applier := session.NewProjectionApplier()
	ev := makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 1, Status: "success", IssueID: "PROJ-1",
	})

	// when
	err := applier.Apply(ev)

	// then
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := applier.State()
	if state.TotalExpeditions != 1 {
		t.Errorf("TotalExpeditions = %d, want 1", state.TotalExpeditions)
	}
	if state.Succeeded != 1 {
		t.Errorf("Succeeded = %d, want 1", state.Succeeded)
	}
}

func TestProjectionApplier_Rebuild_ResetsAndReplays(t *testing.T) {
	// given: applier with pre-existing state
	applier := session.NewProjectionApplier()
	applier.Apply(makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 1, Status: "failed",
	}))
	if applier.State().Failed != 1 {
		t.Fatal("precondition: expected 1 failure before rebuild")
	}

	// when: rebuild with a different set of events
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success",
		}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "success",
		}),
	}
	err := applier.Rebuild(events)

	// then: state reflects only the rebuilt events
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := applier.State()
	if state.TotalExpeditions != 2 {
		t.Errorf("TotalExpeditions = %d, want 2", state.TotalExpeditions)
	}
	if state.Succeeded != 2 {
		t.Errorf("Succeeded = %d, want 2", state.Succeeded)
	}
	if state.Failed != 0 {
		t.Errorf("Failed = %d, want 0 (pre-existing state should be reset)", state.Failed)
	}
}

func TestProjectionApplier_Rebuild_Empty(t *testing.T) {
	// given
	applier := session.NewProjectionApplier()
	applier.Apply(makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
		Expedition: 1, Status: "success",
	}))

	// when: rebuild with empty events
	err := applier.Rebuild(nil)

	// then: state is zeroed
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := applier.State()
	if state.TotalExpeditions != 0 {
		t.Errorf("TotalExpeditions = %d, want 0", state.TotalExpeditions)
	}
}

func TestProjectionApplier_ConsistentWithProjectState(t *testing.T) {
	// given: same events applied via ProjectionApplier and ProjectState
	events := []domain.Event{
		makeProjectionEventAt(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success", IssueID: "P-1",
		}, time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)),
		makeProjectionEventAt(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "failed",
		}, time.Date(2026, 3, 1, 11, 0, 0, 0, time.UTC)),
		makeProjectionEvent(domain.EventGradientChanged, domain.GradientChangedData{Level: 3, Operator: "auto"}),
		makeProjectionEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "spec-1"}),
		makeProjectionEvent(domain.EventGommageTriggered, domain.GommageTriggeredData{Expedition: 3, ConsecutiveFailures: 3}),
	}

	// when
	applier := session.NewProjectionApplier()
	applier.Rebuild(events)
	applierState := applier.State()
	directState := session.ProjectState(events)

	// then: both should produce identical state
	if applierState.TotalExpeditions != directState.TotalExpeditions {
		t.Errorf("TotalExpeditions: applier=%d, direct=%d", applierState.TotalExpeditions, directState.TotalExpeditions)
	}
	if applierState.Succeeded != directState.Succeeded {
		t.Errorf("Succeeded: applier=%d, direct=%d", applierState.Succeeded, directState.Succeeded)
	}
	if applierState.Failed != directState.Failed {
		t.Errorf("Failed: applier=%d, direct=%d", applierState.Failed, directState.Failed)
	}
	if applierState.GradientLevel != directState.GradientLevel {
		t.Errorf("GradientLevel: applier=%d, direct=%d", applierState.GradientLevel, directState.GradientLevel)
	}
	if applierState.GommageCount != directState.GommageCount {
		t.Errorf("GommageCount: applier=%d, direct=%d", applierState.GommageCount, directState.GommageCount)
	}
	if applierState.DMailsStaged != directState.DMailsStaged {
		t.Errorf("DMailsStaged: applier=%d, direct=%d", applierState.DMailsStaged, directState.DMailsStaged)
	}
}

func TestProjectionApplier_SerializeDeserialize_RoundTrip(t *testing.T) {
	// given: applier with accumulated state
	applier := session.NewProjectionApplier()
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success", IssueID: "P-1",
		}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "failed",
		}),
		makeProjectionEvent(domain.EventGradientChanged, domain.GradientChangedData{Level: 3}),
		makeProjectionEvent(domain.EventDMailStaged, domain.DMailStagedData{Name: "spec-1"}),
		makeProjectionEvent(domain.EventGommageTriggered, domain.GommageTriggeredData{Expedition: 3}),
	}
	applier.Rebuild(events)
	original := applier.State()

	// when: serialize then deserialize into a fresh applier
	data, err := applier.Serialize()
	if err != nil {
		t.Fatalf("serialize: %v", err)
	}
	restored := session.NewProjectionApplier()
	if err := restored.Deserialize(data); err != nil {
		t.Fatalf("deserialize: %v", err)
	}

	// then: restored state matches original
	got := restored.State()
	if got.TotalExpeditions != original.TotalExpeditions {
		t.Errorf("TotalExpeditions: got %d, want %d", got.TotalExpeditions, original.TotalExpeditions)
	}
	if got.Succeeded != original.Succeeded {
		t.Errorf("Succeeded: got %d, want %d", got.Succeeded, original.Succeeded)
	}
	if got.Failed != original.Failed {
		t.Errorf("Failed: got %d, want %d", got.Failed, original.Failed)
	}
	if got.GradientLevel != original.GradientLevel {
		t.Errorf("GradientLevel: got %d, want %d", got.GradientLevel, original.GradientLevel)
	}
	if got.GommageCount != original.GommageCount {
		t.Errorf("GommageCount: got %d, want %d", got.GommageCount, original.GommageCount)
	}
	if got.DMailsStaged != original.DMailsStaged {
		t.Errorf("DMailsStaged: got %d, want %d", got.DMailsStaged, original.DMailsStaged)
	}
	if got.ConsecutiveFailures != original.ConsecutiveFailures {
		t.Errorf("ConsecutiveFailures: got %d, want %d", got.ConsecutiveFailures, original.ConsecutiveFailures)
	}
}

func TestProjectionApplier_DeserializeCorrupt(t *testing.T) {
	// given
	applier := session.NewProjectionApplier()

	// when
	err := applier.Deserialize([]byte("not-json"))

	// then
	if err == nil {
		t.Fatal("expected error for corrupt data, got nil")
	}
}

func TestProjectionApplier_SnapshotPlusDelta(t *testing.T) {
	// given: build state from first 3 events, serialize
	events := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 1, Status: "success",
		}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 2, Status: "success",
		}),
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 3, Status: "failed",
		}),
	}
	deltaEvents := []domain.Event{
		makeProjectionEvent(domain.EventExpeditionCompleted, domain.ExpeditionCompletedData{
			Expedition: 4, Status: "success",
		}),
		makeProjectionEvent(domain.EventGradientChanged, domain.GradientChangedData{Level: 5}),
	}

	// Snapshot at event 3
	snapshotApplier := session.NewProjectionApplier()
	snapshotApplier.Rebuild(events)
	snapData, _ := snapshotApplier.Serialize()

	// Full replay for comparison
	fullApplier := session.NewProjectionApplier()
	fullApplier.Rebuild(append(events, deltaEvents...))
	expected := fullApplier.State()

	// when: restore from snapshot + apply delta
	restored := session.NewProjectionApplier()
	restored.Deserialize(snapData)
	for _, ev := range deltaEvents {
		restored.Apply(ev)
	}
	got := restored.State()

	// then: snapshot+delta == full replay
	if got.TotalExpeditions != expected.TotalExpeditions {
		t.Errorf("TotalExpeditions: got %d, want %d", got.TotalExpeditions, expected.TotalExpeditions)
	}
	if got.Succeeded != expected.Succeeded {
		t.Errorf("Succeeded: got %d, want %d", got.Succeeded, expected.Succeeded)
	}
	if got.Failed != expected.Failed {
		t.Errorf("Failed: got %d, want %d", got.Failed, expected.Failed)
	}
	if got.GradientLevel != expected.GradientLevel {
		t.Errorf("GradientLevel: got %d, want %d", got.GradientLevel, expected.GradientLevel)
	}
}
