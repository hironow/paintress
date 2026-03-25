package domain

import "testing"

func TestExpeditionTargetsFromWaves_PendingOnly(t *testing.T) {
	// given: wave with 3 steps, s1 completed
	waves := []WaveProgress{
		{
			WaveID: "auth-w1",
			Title:  "Auth wave",
			Steps: []StepProgress{
				{StepID: "s1", Title: "Middleware", Status: StepCompleted},
				{StepID: "s2", Title: "Session", Status: StepPending, Acceptance: "persist across restarts"},
				{StepID: "s3", Title: "Login", Status: StepFailed},
			},
		},
	}

	// when
	targets := ExpeditionTargetsFromWaves(waves)

	// then
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets (s2 pending, s3 failed), got %d", len(targets))
	}
	if targets[0].ID != "auth-w1:s2" {
		t.Errorf("target[0].ID = %q, want auth-w1:s2", targets[0].ID)
	}
	if targets[0].Acceptance != "persist across restarts" {
		t.Errorf("acceptance not propagated: %q", targets[0].Acceptance)
	}
	if targets[1].ID != "auth-w1:s3" {
		t.Errorf("target[1].ID = %q, want auth-w1:s3", targets[1].ID)
	}
}

func TestExpeditionTargetsFromWaves_AllCompleted(t *testing.T) {
	// given: fully completed wave
	waves := []WaveProgress{
		{
			WaveID: "done-w1",
			Steps:  []StepProgress{{StepID: "s1", Status: StepCompleted}},
		},
	}

	// when
	targets := ExpeditionTargetsFromWaves(waves)

	// then
	if len(targets) != 0 {
		t.Errorf("expected 0 targets for completed wave, got %d", len(targets))
	}
}

func TestExpeditionTargetsFromWaves_SingleStepWave(t *testing.T) {
	// given: single-step wave (stepID == waveID)
	waves := []WaveProgress{
		{
			WaveID: "fix-w1",
			Title:  "Quick fix",
			Steps:  []StepProgress{{StepID: "fix-w1", Title: "Quick fix", Status: StepPending}},
		},
	}

	// when
	targets := ExpeditionTargetsFromWaves(waves)

	// then
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].ID != "fix-w1" {
		t.Errorf("single-step target ID = %q, want wave ID only", targets[0].ID)
	}
	if !targets[0].IsWaveTarget() {
		t.Error("should be wave target")
	}
}

func TestExpeditionTargetsFromWaves_MultipleWaves(t *testing.T) {
	// given: 2 waves, mixed status
	waves := []WaveProgress{
		{WaveID: "a", Steps: []StepProgress{{StepID: "a1", Status: StepPending}}},
		{WaveID: "b", Steps: []StepProgress{{StepID: "b1", Status: StepCompleted}, {StepID: "b2", Status: StepPending}}},
	}

	// when
	targets := ExpeditionTargetsFromWaves(waves)

	// then: a1 + b2
	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}
	if targets[0].ID != "a:a1" {
		t.Errorf("target[0].ID = %q", targets[0].ID)
	}
	if targets[1].ID != "b:b2" {
		t.Errorf("target[1].ID = %q", targets[1].ID)
	}
}
