package domain

import "testing"

func TestProjectWaveState_EmptyArchive(t *testing.T) {
	// when
	result := ProjectWaveState(nil)

	// then
	if len(result) != 0 {
		t.Errorf("expected 0 waves, got %d", len(result))
	}
}

func TestProjectWaveState_SpecOnly_AllPending(t *testing.T) {
	// given
	dmails := []DMail{
		{
			Kind:        "specification",
			Description: "Auth wave",
			Wave: &WaveReference{
				ID: "auth-w1",
				Steps: []WaveStepDef{
					{ID: "s1", Title: "Add middleware", Acceptance: "intercepts /api/*"},
					{ID: "s2", Title: "Add login"},
				},
			},
		},
	}

	// when
	result := ProjectWaveState(dmails)

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
		if s.Status != StepPending {
			t.Errorf("step %s: status = %q, want pending", s.StepID, s.Status)
		}
	}
	if w.Steps[0].Acceptance != "intercepts /api/*" {
		t.Errorf("acceptance = %q, want propagated from WaveStepDef", w.Steps[0].Acceptance)
	}
}

func TestProjectWaveState_StepCompleted(t *testing.T) {
	// given
	dmails := []DMail{
		{
			Kind:        "specification",
			Description: "Auth wave",
			Wave:        &WaveReference{ID: "auth-w1", Steps: []WaveStepDef{{ID: "s1", Title: "Step 1"}, {ID: "s2", Title: "Step 2"}}},
		},
		{
			Kind: "report",
			Wave: &WaveReference{ID: "auth-w1", Step: "s1"},
			// severity empty = success
		},
	}

	// when
	result := ProjectWaveState(dmails)

	// then
	w := result[0]
	if w.Steps[0].Status != StepCompleted {
		t.Errorf("s1 status = %q, want completed", w.Steps[0].Status)
	}
	if w.Steps[1].Status != StepPending {
		t.Errorf("s2 status = %q, want pending", w.Steps[1].Status)
	}
	if w.IsComplete() {
		t.Error("wave should not be complete (s2 still pending)")
	}
}

func TestProjectWaveState_AllStepsCompleted(t *testing.T) {
	// given
	dmails := []DMail{
		{Kind: "specification", Wave: &WaveReference{ID: "w1", Steps: []WaveStepDef{{ID: "s1"}, {ID: "s2"}}}},
		{Kind: "report", Wave: &WaveReference{ID: "w1", Step: "s1"}},
		{Kind: "report", Wave: &WaveReference{ID: "w1", Step: "s2"}},
	}

	// when
	result := ProjectWaveState(dmails)

	// then
	if !result[0].IsComplete() {
		t.Error("wave should be complete (all steps done)")
	}
}

func TestProjectWaveState_FailedThenRetrySuccess(t *testing.T) {
	// given: fail → feedback → retry success
	dmails := []DMail{
		{Kind: "specification", Wave: &WaveReference{ID: "w1", Steps: []WaveStepDef{{ID: "s1"}}}},
		{Kind: "report", Severity: "high", Wave: &WaveReference{ID: "w1", Step: "s1"}},
		{Kind: "implementation-feedback", Wave: &WaveReference{ID: "w1", Step: "s1"}},
		{Kind: "report", Wave: &WaveReference{ID: "w1", Step: "s1"}}, // retry success
	}

	// when
	result := ProjectWaveState(dmails)

	// then
	s := result[0].Steps[0]
	if s.Status != StepCompleted {
		t.Errorf("status = %q, want completed after retry", s.Status)
	}
	if s.Attempts != 2 {
		t.Errorf("attempts = %d, want 2", s.Attempts)
	}
}

func TestProjectWaveState_SingleStepWave(t *testing.T) {
	// given: wave with no explicit steps = wave itself is the step
	dmails := []DMail{
		{Kind: "specification", Description: "Small fix", Wave: &WaveReference{ID: "fix-w1"}},
		{Kind: "report", Wave: &WaveReference{ID: "fix-w1"}}, // step="" → uses waveID
	}

	// when
	result := ProjectWaveState(dmails)

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
	dmails := []DMail{
		{Kind: "specification", Description: "Wave A", Wave: &WaveReference{ID: "a", Steps: []WaveStepDef{{ID: "a1"}}}},
		{Kind: "specification", Description: "Wave B", Wave: &WaveReference{ID: "b", Steps: []WaveStepDef{{ID: "b1"}, {ID: "b2"}}}},
		{Kind: "report", Wave: &WaveReference{ID: "a", Step: "a1"}},
	}

	// when
	result := ProjectWaveState(dmails)

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
	dmails := []DMail{
		{Kind: "report", Description: "Legacy report"}, // no wave field
		{Kind: "specification", Wave: &WaveReference{ID: "w1", Steps: []WaveStepDef{{ID: "s1"}}}},
		{Kind: "implementation-feedback", Description: "Legacy feedback"}, // no wave field
	}

	// when
	result := ProjectWaveState(dmails)

	// then
	if len(result) != 1 {
		t.Fatalf("expected 1 wave (non-wave D-Mails ignored), got %d", len(result))
	}
}

func TestProjectWaveState_ReportWithoutSpec_Ignored(t *testing.T) {
	// given: report for unknown wave
	dmails := []DMail{
		{Kind: "report", Wave: &WaveReference{ID: "unknown", Step: "s1"}},
	}

	// when
	result := ProjectWaveState(dmails)

	// then
	if len(result) != 0 {
		t.Errorf("expected 0 waves (report without spec), got %d", len(result))
	}
}
