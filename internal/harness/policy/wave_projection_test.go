package policy_test

// Tests ProjectWaveState pure function — archive D-Mail to wave progress projection

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/harness/policy"
)

func TestProjectWaveState_EmptyArchive(t *testing.T) {
	// when
	result := policy.ProjectWaveState(nil)

	// then
	if len(result) != 0 {
		t.Errorf("expected 0 waves, got %d", len(result))
	}
}

func TestProjectWaveState_SpecOnly_AllPending(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{
			Kind:        "specification",
			Description: "Auth wave",
			Wave: &domain.WaveReference{
				ID: "auth-w1",
				Steps: []domain.WaveStepDef{
					{ID: "s1", Title: "Add middleware", Acceptance: "intercepts /api/*"},
					{ID: "s2", Title: "Add login"},
				},
			},
		},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	if len(result) != 1 {
		t.Fatalf("expected 1 wave, got %d", len(result))
	}
	w := result[0]
	if w.WaveID != "auth-w1" {
		t.Errorf("wave ID = %q, want auth-w1", w.WaveID)
	}
	if len(w.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(w.Steps))
	}
	for _, s := range w.Steps {
		if s.Status != domain.StepPending {
			t.Errorf("step %s: status = %q, want pending", s.StepID, s.Status)
		}
	}
	if w.Steps[0].Acceptance != "intercepts /api/*" {
		t.Errorf("acceptance = %q, want propagated from WaveStepDef", w.Steps[0].Acceptance)
	}
}

func TestProjectWaveState_StepCompleted(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{
			Kind:        "specification",
			Description: "Auth wave",
			Wave:        &domain.WaveReference{ID: "auth-w1", Steps: []domain.WaveStepDef{{ID: "s1", Title: "Step 1"}, {ID: "s2", Title: "Step 2"}}},
		},
		{
			Kind: "report",
			Wave: &domain.WaveReference{ID: "auth-w1", Step: "s1"},
			// severity empty = success
		},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	w := result[0]
	if w.Steps[0].Status != domain.StepCompleted {
		t.Errorf("s1 status = %q, want completed", w.Steps[0].Status)
	}
	if w.Steps[1].Status != domain.StepPending {
		t.Errorf("s2 status = %q, want pending", w.Steps[1].Status)
	}
	if w.IsComplete() {
		t.Error("wave should not be complete (s2 still pending)")
	}
}

func TestProjectWaveState_AllStepsCompleted(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{Kind: "specification", Wave: &domain.WaveReference{ID: "w1", Steps: []domain.WaveStepDef{{ID: "s1"}, {ID: "s2"}}}},
		{Kind: "report", Wave: &domain.WaveReference{ID: "w1", Step: "s1"}},
		{Kind: "report", Wave: &domain.WaveReference{ID: "w1", Step: "s2"}},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	if !result[0].IsComplete() {
		t.Error("wave should be complete (all steps done)")
	}
}

func TestProjectWaveState_FailedThenRetrySuccess(t *testing.T) {
	// given: fail → feedback → retry success
	dmails := []domain.DMail{
		{Kind: "specification", Wave: &domain.WaveReference{ID: "w1", Steps: []domain.WaveStepDef{{ID: "s1"}}}},
		{Kind: "report", Severity: "high", Wave: &domain.WaveReference{ID: "w1", Step: "s1"}},
		{Kind: "implementation-feedback", Wave: &domain.WaveReference{ID: "w1", Step: "s1"}},
		{Kind: "report", Wave: &domain.WaveReference{ID: "w1", Step: "s1"}}, // retry success
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	s := result[0].Steps[0]
	if s.Status != domain.StepCompleted {
		t.Errorf("status = %q, want completed after retry", s.Status)
	}
	if s.Attempts != 2 {
		t.Errorf("attempts = %d, want 2", s.Attempts)
	}
}

func TestProjectWaveState_SingleStepWave(t *testing.T) {
	// given: wave with no explicit steps = wave itself is the step
	dmails := []domain.DMail{
		{Kind: "specification", Description: "Small fix", Wave: &domain.WaveReference{ID: "fix-w1"}},
		{Kind: "report", Wave: &domain.WaveReference{ID: "fix-w1"}}, // step="" → uses waveID
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	if len(result) != 1 {
		t.Fatalf("expected 1 wave, got %d", len(result))
	}
	w := result[0]
	if len(w.Steps) != 1 {
		t.Fatalf("expected 1 implicit step, got %d", len(w.Steps))
	}
	if w.Steps[0].StepID != "fix-w1" {
		t.Errorf("implicit step ID = %q, want wave ID", w.Steps[0].StepID)
	}
	if !w.IsComplete() {
		t.Error("single-step wave should be complete")
	}
}

func TestProjectWaveState_MultipleWaves(t *testing.T) {
	// given
	dmails := []domain.DMail{
		{Kind: "specification", Description: "Wave A", Wave: &domain.WaveReference{ID: "a", Steps: []domain.WaveStepDef{{ID: "a1"}}}},
		{Kind: "specification", Description: "Wave B", Wave: &domain.WaveReference{ID: "b", Steps: []domain.WaveStepDef{{ID: "b1"}, {ID: "b2"}}}},
		{Kind: "report", Wave: &domain.WaveReference{ID: "a", Step: "a1"}},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	if len(result) != 2 {
		t.Fatalf("expected 2 waves, got %d", len(result))
	}
	if !result[0].IsComplete() {
		t.Error("wave A should be complete")
	}
	if result[1].IsComplete() {
		t.Error("wave B should not be complete")
	}
}

func TestProjectWaveState_DMailsWithoutWaveIgnored(t *testing.T) {
	// given: mix of wave and non-wave D-Mails
	dmails := []domain.DMail{
		{Kind: "report", Description: "Legacy report"}, // no wave field
		{Kind: "specification", Wave: &domain.WaveReference{ID: "w1", Steps: []domain.WaveStepDef{{ID: "s1"}}}},
		{Kind: "implementation-feedback", Description: "Legacy feedback"}, // no wave field
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	if len(result) != 1 {
		t.Fatalf("expected 1 wave (non-wave D-Mails ignored), got %d", len(result))
	}
}

func TestProjectWaveState_SkippedReportCompletesStep(t *testing.T) {
	// given: spec + skipped report (Claude determined issue was already done)
	dmails := []domain.DMail{
		{
			Kind: "specification",
			Wave: &domain.WaveReference{ID: "auth-w1", Steps: []domain.WaveStepDef{{ID: "s1", Title: "Step 1"}, {ID: "s2", Title: "Step 2"}}},
		},
		{
			Kind:     "report",
			Wave:     &domain.WaveReference{ID: "auth-w1", Step: "s1"},
			Metadata: map[string]string{"status": "skipped"},
			// severity empty → currently treated as success (completed)
		},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then: s1 should be completed (skipped = already done)
	w := result[0]
	if w.Steps[0].Status != domain.StepCompleted {
		t.Errorf("s1 status = %q, want completed (skipped should mark as completed)", w.Steps[0].Status)
	}
	if w.Steps[1].Status != domain.StepPending {
		t.Errorf("s2 status = %q, want pending", w.Steps[1].Status)
	}
}

func TestProjectWaveState_MultiWaveProgression(t *testing.T) {
	// given: 2 waves with multiple steps, some completed/skipped
	// Simulates go-taskboard state after several expeditions
	dmails := []domain.DMail{
		// Wave 1: spec with 3 steps
		{Kind: "specification", Wave: &domain.WaveReference{
			ID:    "validation:w1",
			Steps: []domain.WaveStepDef{{ID: "2"}, {ID: "3"}, {ID: "4"}},
		}},
		// Wave 2: spec with 2 steps
		{Kind: "specification", Wave: &domain.WaveReference{
			ID:    "api:w1",
			Steps: []domain.WaveStepDef{{ID: "6"}, {ID: "7"}},
		}},
		// Wave 1, step 2: completed via skipped report (severity empty)
		{Kind: "report", Wave: &domain.WaveReference{ID: "validation:w1", Step: "2"}},
		// Wave 1, step 3: completed via success report
		{Kind: "report", Severity: "low", Wave: &domain.WaveReference{ID: "validation:w1", Step: "3"}},
	}

	// when
	result := policy.ProjectWaveState(dmails)
	targets := policy.ExpeditionTargetsFromWaves(result)

	// then: wave 1 has 1 pending (step 4), wave 2 has 2 pending (steps 6, 7)
	if len(targets) != 3 {
		t.Fatalf("expected 3 pending targets, got %d", len(targets))
	}
	if targets[0].StepID != "4" {
		t.Errorf("targets[0].StepID = %q, want 4", targets[0].StepID)
	}
	if targets[1].StepID != "6" {
		t.Errorf("targets[1].StepID = %q, want 6", targets[1].StepID)
	}
	if targets[2].StepID != "7" {
		t.Errorf("targets[2].StepID = %q, want 7", targets[2].StepID)
	}
}

func TestProjectWaveState_ReportBeforeSpec_StillCompletes(t *testing.T) {
	// given: report arrives before spec in archive sort order
	// (pt-report-* sorts before spec-* alphabetically)
	// This is the real go-taskboard scenario causing the infinite loop
	dmails := []domain.DMail{
		// Report first (alphabetical file order: pt- < spec-)
		{Kind: "report", Wave: &domain.WaveReference{ID: "validation:w1", Step: "2"}},
		// Spec second
		{Kind: "specification", Wave: &domain.WaveReference{
			ID:    "validation:w1",
			Steps: []domain.WaveStepDef{{ID: "2"}, {ID: "3"}},
		}},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then: step 2 should be completed even though report came before spec
	if len(result) != 1 {
		t.Fatalf("expected 1 wave, got %d", len(result))
	}
	w := result[0]
	if w.Steps[0].Status != domain.StepCompleted {
		t.Errorf("step 2 status = %q, want completed (report-before-spec)", w.Steps[0].Status)
	}
	if w.Steps[1].Status != domain.StepPending {
		t.Errorf("step 3 status = %q, want pending", w.Steps[1].Status)
	}
}

func TestProjectWaveState_ParsedReportCompletesStep(t *testing.T) {
	// given: parse actual D-Mail bytes to verify real-world round-trip
	specBytes := []byte(`---
name: spec-validation-w1
kind: specification
dmail-schema-version: "1"
wave:
  id: "入力バリデーション:cluster-w1"
  steps:
    - id: "2"
      title: Add validation
    - id: "3"
      title: Add error handling
---
`)
	reportBytes := []byte(`---
name: pt-report-cluster-w1-2_07ed50df
kind: report
description: 'Expedition #4 completed implement for 2'
issues:
    - "2"
dmail-schema-version: "1"
wave:
    id: "入力バリデーション:cluster-w1"
    step: "2"
---
# Report
`)
	spec, err := domain.ParseDMail(specBytes)
	if err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	report, err := domain.ParseDMail(reportBytes)
	if err != nil {
		t.Fatalf("parse report: %v", err)
	}

	// when
	result := policy.ProjectWaveState([]domain.DMail{spec, report})

	// then: step 2 completed
	w := result[0]
	if w.Steps[0].StepID != "2" || w.Steps[0].Status != domain.StepCompleted {
		t.Errorf("step 2: status=%q, want completed", w.Steps[0].Status)
	}
	if w.Steps[1].StepID != "3" || w.Steps[1].Status != domain.StepPending {
		t.Errorf("step 3: status=%q, want pending", w.Steps[1].Status)
	}
}

func TestProjectWaveState_DuplicateStepIDs_ReportCompletesStep(t *testing.T) {
	// given: spec with duplicate step IDs (real go-taskboard data)
	// ProjectWaveState uses map[stepID], so duplicates are deduplicated (last wins)
	dmails := []domain.DMail{
		{
			Kind: "specification",
			Wave: &domain.WaveReference{
				ID: "入力バリデーション:cluster-w1",
				Steps: []domain.WaveStepDef{
					{ID: "2", Title: "First occurrence of step 2"},
					{ID: "3", Title: "Step 3"},
					{ID: "2", Title: "Second occurrence of step 2"},
					{ID: "5", Title: "Step 5"},
					{ID: "2", Title: "Third occurrence of step 2"},
				},
			},
		},
		{
			Kind: "report",
			Wave: &domain.WaveReference{ID: "入力バリデーション:cluster-w1", Step: "2"},
			// severity empty = StepCompleted
		},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then: step 2 completed despite duplicates, steps 3 and 5 pending
	w := result[0]
	for _, s := range w.Steps {
		if s.StepID == "2" && s.Status != domain.StepCompleted {
			t.Errorf("step 2 status = %q, want completed", s.Status)
		}
		if (s.StepID == "3" || s.StepID == "5") && s.Status != domain.StepPending {
			t.Errorf("step %s status = %q, want pending", s.StepID, s.Status)
		}
	}

	// Pending should be 3 and 5 only
	pending := w.PendingSteps()
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending steps, got %d", len(pending))
	}
}

func TestProjectWaveState_ReportWithoutSpec_Ignored(t *testing.T) {
	// given: report for unknown wave
	dmails := []domain.DMail{
		{Kind: "report", Wave: &domain.WaveReference{ID: "unknown", Step: "s1"}},
	}

	// when
	result := policy.ProjectWaveState(dmails)

	// then
	if len(result) != 0 {
		t.Errorf("expected 0 waves (report without spec), got %d", len(result))
	}
}
