package usecase

// white-box-reason: tests waveTargetProvider using unexported struct and verifies inbox+archive merge logic

import (
	"context"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

type fakeArchiveReader struct {
	dmails []domain.DMail
}

func (f *fakeArchiveReader) ReadArchiveDMails(_ context.Context) ([]domain.DMail, error) {
	return f.dmails, nil
}

type fakeInboxReader struct {
	dmails []domain.DMail
}

func (f *fakeInboxReader) ReadInboxDMails(_ context.Context) ([]domain.DMail, error) {
	return f.dmails, nil
}

func TestWaveTargetProvider_EmptyArchive_InboxSpecsProvideTargets(t *testing.T) {
	// given: archive is empty (initial state), inbox has spec with wave steps
	archive := &fakeArchiveReader{dmails: nil}
	inbox := &fakeInboxReader{dmails: []domain.DMail{
		{
			Name: "spec-auth-w1",
			Kind: "specification",
			Wave: &domain.WaveReference{
				ID: "auth:w1",
				Steps: []domain.WaveStepDef{
					{ID: "s1", Title: "Add JWT middleware", Acceptance: "Middleware intercepts /api/*"},
					{ID: "s2", Title: "Add token validation", Acceptance: "Tokens validated on every request"},
				},
			},
		},
	}}

	// when
	provider := NewWaveTargetProvider(archive, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: targets should include the inbox spec's steps
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets from inbox spec, got %d", len(targets))
	}
	if targets[0].StepID != "s1" {
		t.Errorf("targets[0].StepID = %q, want s1", targets[0].StepID)
	}
	if targets[1].StepID != "s2" {
		t.Errorf("targets[1].StepID = %q, want s2", targets[1].StepID)
	}
}

func TestWaveTargetProvider_SkippedStepProgressesToNext(t *testing.T) {
	// given: archive has spec + skipped report for step s1
	// Simulates: expedition ran, Claude said "already done", skipped report archived
	archive := &fakeArchiveReader{
		dmails: []domain.DMail{
			{
				Name: "spec-auth-w1",
				Kind: "specification",
				Wave: &domain.WaveReference{
					ID: "validation:w1",
					Steps: []domain.WaveStepDef{
						{ID: "2", Title: "Add validation"},
						{ID: "3", Title: "Add error handling"},
						{ID: "4", Title: "Add tests"},
					},
				},
			},
			{
				// Skipped report: severity empty = StepCompleted
				Name:     "pt-report-validation-w1-2_00000000",
				Kind:     "report",
				Severity: "", // forced empty for skipped
				Wave:     &domain.WaveReference{ID: "validation:w1", Step: "2"},
			},
		},
	}
	inbox := &fakeInboxReader{dmails: nil}

	// when
	provider := NewWaveTargetProvider(archive, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: step 2 completed (skipped), steps 3 and 4 still pending
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 2 {
		t.Fatalf("expected 2 pending targets (steps 3,4), got %d", len(targets))
	}
	if targets[0].StepID != "3" {
		t.Errorf("targets[0].StepID = %q, want 3", targets[0].StepID)
	}
	if targets[1].StepID != "4" {
		t.Errorf("targets[1].StepID = %q, want 4", targets[1].StepID)
	}
}

func TestWaveTargetProvider_MultiWaveWithMixedProgress(t *testing.T) {
	// given: multiple waves from go-taskboard-like state
	// Wave 1: 3 steps, step 2 skipped (completed), steps 3,4 pending
	// Wave 2: 2 steps, all pending
	// Wave 3: 1 step, completed via success report
	archive := &fakeArchiveReader{
		dmails: []domain.DMail{
			// Wave 1 spec
			{Kind: "specification", Wave: &domain.WaveReference{
				ID:    "validation:w1",
				Steps: []domain.WaveStepDef{{ID: "2"}, {ID: "3"}, {ID: "4"}},
			}},
			// Wave 2 spec
			{Kind: "specification", Wave: &domain.WaveReference{
				ID:    "api:w1",
				Steps: []domain.WaveStepDef{{ID: "6"}, {ID: "7"}},
			}},
			// Wave 3 spec (single step, completed)
			{Kind: "specification", Wave: &domain.WaveReference{
				ID:    "pagination:w1",
				Steps: []domain.WaveStepDef{{ID: "1"}},
			}},
			// Wave 1, step 2: skipped report (severity empty)
			{Kind: "report", Severity: "", Wave: &domain.WaveReference{ID: "validation:w1", Step: "2"}},
			// Wave 3, step 1: success report
			{Kind: "report", Severity: "low", Wave: &domain.WaveReference{ID: "pagination:w1", Step: "1"}},
		},
	}
	inbox := &fakeInboxReader{dmails: nil}

	// when
	provider := NewWaveTargetProvider(archive, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: 4 pending targets across wave 1 (steps 3,4) and wave 2 (steps 6,7)
	// wave 3 is fully completed
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 4 {
		t.Fatalf("expected 4 pending targets, got %d", len(targets))
	}
	// Wave 1 pending: 3, 4
	if targets[0].StepID != "3" || targets[0].WaveID != "validation:w1" {
		t.Errorf("targets[0] = %s:%s, want validation:w1:3", targets[0].WaveID, targets[0].StepID)
	}
	if targets[1].StepID != "4" || targets[1].WaveID != "validation:w1" {
		t.Errorf("targets[1] = %s:%s, want validation:w1:4", targets[1].WaveID, targets[1].StepID)
	}
	// Wave 2 pending: 6, 7
	if targets[2].StepID != "6" || targets[2].WaveID != "api:w1" {
		t.Errorf("targets[2] = %s:%s, want api:w1:6", targets[2].WaveID, targets[2].StepID)
	}
	if targets[3].StepID != "7" || targets[3].WaveID != "api:w1" {
		t.Errorf("targets[3] = %s:%s, want api:w1:7", targets[3].WaveID, targets[3].StepID)
	}
}

func TestWaveTargetProvider_AllWavesCompleted_NoTargets(t *testing.T) {
	// given: all waves fully completed (mix of success and skipped)
	archive := &fakeArchiveReader{
		dmails: []domain.DMail{
			{Kind: "specification", Wave: &domain.WaveReference{
				ID:    "w1",
				Steps: []domain.WaveStepDef{{ID: "s1"}, {ID: "s2"}},
			}},
			{Kind: "report", Severity: "", Wave: &domain.WaveReference{ID: "w1", Step: "s1"}},   // skipped
			{Kind: "report", Severity: "low", Wave: &domain.WaveReference{ID: "w1", Step: "s2"}}, // success
		},
	}
	inbox := &fakeInboxReader{dmails: nil}

	// when
	provider := NewWaveTargetProvider(archive, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: no pending targets
	if err != nil {
		t.Fatalf("FetchTargets error: %v", err)
	}
	if len(targets) != 0 {
		t.Errorf("expected 0 pending targets (all completed), got %d", len(targets))
	}
}

func TestWaveTargetProvider_ArchiveCompletedStep_InboxSpecExcludesIt(t *testing.T) {
	// given: archive has spec + report marking s1 as completed
	archive := &fakeArchiveReader{
		dmails: []domain.DMail{
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
			{
				Name: "pt-report-auth-w1-s1_00000000",
				Kind: "report",
				Wave: &domain.WaveReference{ID: "auth:w1", Step: "s1"},
			},
		},
	}

	// inbox has the same spec (still in inbox, not yet archived)
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
	provider := NewWaveTargetProvider(archive, inbox)
	targets, err := provider.FetchTargets(context.Background())

	// then: only s2 is pending (s1 completed via archive report)
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
