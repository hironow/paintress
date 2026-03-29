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
