package usecase

// white-box-reason: tests waveTargetProvider using unexported struct and verifies event-sourced step progress + inbox fallback

import (
	"context"
	"testing"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

var fixedTime = time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)

type fakeStepProgressReader struct {
	progress *domain.WaveStepProgress
	err      error
}

func (f *fakeStepProgressReader) ReadStepProgress(_ context.Context) (*domain.WaveStepProgress, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.progress, nil
}

type fakeInboxReader struct {
	dmails []domain.DMail
}

func (f *fakeInboxReader) ReadInboxDMails(_ context.Context) ([]domain.DMail, error) {
	return f.dmails, nil
}

func TestWaveTargetProvider_EventSourcedProgress(t *testing.T) {
	// given: StepProgressReader returns progress with 2 waves, 1 step completed
	events := []domain.Event{
		specEvent("auth-w1", []domain.WaveStepDef{
			{ID: "s1", Title: "Add JWT middleware"},
			{ID: "s2", Title: "Add token validation"},
		}),
		completedEvent("success", "auth-w1", "s1"),
	}
	progress := domain.ProjectWaveStepProgress(events)
	reader := &fakeStepProgressReader{progress: progress}

	// when
	provider := NewWaveTargetProvider(reader, nil)
	targets, err := provider.FetchTargets(context.Background())

	// then: only s2 pending
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 pending target, got %d", len(targets))
	}
	if targets[0].StepID != "s2" {
		t.Errorf("target.StepID = %q, want s2", targets[0].StepID)
	}
}

func TestWaveTargetProvider_FallbackToInbox(t *testing.T) {
	// given: empty progress (no spec.registered events) + inbox has spec
	emptyProgress := domain.ProjectWaveStepProgress(nil)
	reader := &fakeStepProgressReader{progress: emptyProgress}
	inbox := &fakeInboxReader{dmails: []domain.DMail{
		{
			Name: "spec-auth-w1",
			Kind: "specification",
			Wave: &domain.WaveReference{
				ID: "auth:w1",
				Steps: []domain.WaveStepDef{
					{ID: "s1", Title: "Add JWT middleware"},
					{ID: "s2", Title: "Add token validation"},
				},
			},
		},
	}}

	// when
	provider := NewWaveTargetProvider(reader, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: fallback to inbox, 2 pending targets
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets from inbox fallback, got %d", len(targets))
	}
}

func TestWaveTargetProvider_AllCompleted_NoTargets(t *testing.T) {
	// given: all steps completed
	events := []domain.Event{
		specEvent("w1", []domain.WaveStepDef{
			{ID: "s1", Title: "Step 1"},
		}),
		completedEvent("success", "w1", "s1"),
	}
	progress := domain.ProjectWaveStepProgress(events)
	reader := &fakeStepProgressReader{progress: progress}

	// when
	provider := NewWaveTargetProvider(reader, nil)
	targets, err := provider.FetchTargets(context.Background())

	// then
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets, got %d", len(targets))
	}
}

func TestWaveTargetProvider_WavelessFeedback_BecomesStandaloneTarget(t *testing.T) {
	// given: waves exist (so fallback is not used) + inbox has waveless impl-feedback
	events := []domain.Event{
		specEvent("w1", []domain.WaveStepDef{
			{ID: "s1", Title: "Step 1"},
		}),
		completedEvent("success", "w1", "s1"), // all wave steps completed
	}
	progress := domain.ProjectWaveStepProgress(events)
	reader := &fakeStepProgressReader{progress: progress}
	inbox := &fakeInboxReader{dmails: []domain.DMail{
		{
			Name:        "am-conflict-#30-3fade125",
			Kind:        "implementation-feedback",
			Description: "PR #30 has merge conflicts — rebase needed",
			Severity:    "medium",
			Metadata: map[string]string{
				"pr_number":       "#30",
				"conflict_reason": "CONFLICTING",
				"type":            "merge-conflict",
			},
			Body: "PR #30 has merge conflicts with the base branch.",
			// Wave is nil — this is a waveless feedback D-Mail
		},
	}}

	// when
	provider := NewWaveTargetProvider(reader, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: the waveless feedback becomes a standalone target
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 standalone target from waveless feedback, got %d", len(targets))
	}
	if targets[0].ID != "am-conflict-#30-3fade125" {
		t.Errorf("target.ID = %q, want am-conflict-#30-3fade125", targets[0].ID)
	}
	if targets[0].IsWaveTarget() {
		t.Error("standalone target should not be a wave target")
	}
}

func TestWaveTargetProvider_WavelessFeedback_MixedWithWaveTargets(t *testing.T) {
	// given: 1 pending wave step + 1 waveless feedback in inbox
	events := []domain.Event{
		specEvent("w1", []domain.WaveStepDef{
			{ID: "s1", Title: "Step 1"},
			{ID: "s2", Title: "Step 2"},
		}),
		completedEvent("success", "w1", "s1"),
	}
	progress := domain.ProjectWaveStepProgress(events)
	reader := &fakeStepProgressReader{progress: progress}
	inbox := &fakeInboxReader{dmails: []domain.DMail{
		{
			Name:        "am-conflict-#14-a9c5e6e3",
			Kind:        "implementation-feedback",
			Description: "PR #14 has merge conflicts",
			Metadata:    map[string]string{"type": "merge-conflict"},
			Body:        "Rebase needed.",
		},
	}}

	// when
	provider := NewWaveTargetProvider(reader, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: wave target s2 + standalone feedback target
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets (1 wave + 1 standalone), got %d", len(targets))
	}
}

func TestWaveTargetProvider_SpecificationDMail_NotStandaloneTarget(t *testing.T) {
	// given: inbox has specification D-Mail without Wave (should NOT become standalone)
	events := []domain.Event{
		specEvent("w1", []domain.WaveStepDef{
			{ID: "s1", Title: "Step 1"},
		}),
		completedEvent("success", "w1", "s1"),
	}
	progress := domain.ProjectWaveStepProgress(events)
	reader := &fakeStepProgressReader{progress: progress}
	inbox := &fakeInboxReader{dmails: []domain.DMail{
		{
			Name: "spec-something",
			Kind: "specification",
			Body: "Some spec.",
		},
	}}

	// when
	provider := NewWaveTargetProvider(reader, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: no targets (specification without Wave is not a standalone target)
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 targets (spec is not standalone), got %d", len(targets))
	}
}

// --- test helpers ---

func specEvent(waveID string, steps []domain.WaveStepDef) domain.Event {
	data := domain.SpecRegisteredData{WaveID: waveID, Steps: steps, Source: "test"}
	ev, _ := domain.NewEvent(domain.EventSpecRegistered, data, fixedTime)
	return ev
}

func completedEvent(status, waveID, stepID string) domain.Event {
	data := domain.ExpeditionCompletedData{
		Expedition: 1, Status: status, WaveID: waveID, StepID: stepID,
	}
	ev, _ := domain.NewEvent(domain.EventExpeditionCompleted, data, fixedTime)
	return ev
}
