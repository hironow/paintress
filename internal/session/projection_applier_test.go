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
