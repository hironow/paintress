package domain

// white-box-reason: tests ProjectWaveStepProgress pure function with unexported waveStepEntry internals

import (
	"encoding/json"
	"testing"
	"time"
)

func specEvent(waveID string, steps []WaveStepDef, source string) Event {
	data, _ := json.Marshal(SpecRegisteredData{
		WaveID: waveID,
		Steps:  steps,
		Source: source,
	})
	return Event{Type: EventSpecRegistered, Data: data, Timestamp: time.Now()}
}

func completedEvent(status, waveID, stepID string) Event {
	data, _ := json.Marshal(ExpeditionCompletedData{
		Expedition: 1,
		Status:     status,
		WaveID:     waveID,
		StepID:     stepID,
	})
	return Event{Type: EventExpeditionCompleted, Data: data, Timestamp: time.Now()}
}

func TestProjectWaveStepProgress_SpecRegistered(t *testing.T) {
	// given: single spec.registered event with 2 steps
	events := []Event{
		specEvent("auth-w1", []WaveStepDef{
			{ID: "s1", Title: "Implement handler"},
			{ID: "s2", Title: "Add tests"},
		}, "sj-spec-auth-w1_abc123.md"),
	}

	// when
	progress := ProjectWaveStepProgress(events)

	// then: both steps should be pending targets
	targets := progress.PendingTargets()
	if len(targets) != 2 {
		t.Fatalf("expected 2 pending targets, got %d", len(targets))
	}
	if targets[0].StepID != "s1" {
		t.Errorf("target[0].StepID = %q, want s1", targets[0].StepID)
	}
	if targets[1].StepID != "s2" {
		t.Errorf("target[1].StepID = %q, want s2", targets[1].StepID)
	}
}

func TestProjectWaveStepProgress_CompletedExcludes(t *testing.T) {
	// given: spec registered + step s1 completed
	events := []Event{
		specEvent("auth-w1", []WaveStepDef{
			{ID: "s1", Title: "Implement handler"},
			{ID: "s2", Title: "Add tests"},
		}, "sj-spec-auth-w1.md"),
		completedEvent("success", "auth-w1", "s1"),
	}

	// when
	progress := ProjectWaveStepProgress(events)

	// then: only s2 should be pending
	targets := progress.PendingTargets()
	if len(targets) != 1 {
		t.Fatalf("expected 1 pending target, got %d", len(targets))
	}
	if targets[0].StepID != "s2" {
		t.Errorf("target[0].StepID = %q, want s2", targets[0].StepID)
	}
}

func TestProjectWaveStepProgress_SkippedExcludes(t *testing.T) {
	// given: spec registered + step s1 skipped (skipped = already done)
	events := []Event{
		specEvent("fix-w1", []WaveStepDef{
			{ID: "s1", Title: "Fix bug"},
		}, "sj-spec-fix-w1.md"),
		completedEvent("skipped", "fix-w1", "s1"),
	}

	// when
	progress := ProjectWaveStepProgress(events)

	// then: no pending targets (skipped = completed)
	targets := progress.PendingTargets()
	if len(targets) != 0 {
		t.Errorf("expected 0 pending targets for skipped step, got %d", len(targets))
	}
}

func TestProjectWaveStepProgress_FailedRetains(t *testing.T) {
	// given: spec registered + step s1 failed (should remain pending for retry)
	events := []Event{
		specEvent("core-w1", []WaveStepDef{
			{ID: "s1", Title: "Implement feature"},
			{ID: "s2", Title: "Add tests"},
		}, "sj-spec-core-w1.md"),
		completedEvent("failed", "core-w1", "s1"),
	}

	// when
	progress := ProjectWaveStepProgress(events)

	// then: both steps should still be pending (failed = retry-able)
	targets := progress.PendingTargets()
	if len(targets) != 2 {
		t.Fatalf("expected 2 pending targets (failed retains), got %d", len(targets))
	}
}

func TestProjectWaveStepProgress_DuplicateSpecFirstWins(t *testing.T) {
	// given: same wave registered twice (idempotency)
	events := []Event{
		specEvent("auth-w1", []WaveStepDef{
			{ID: "s1", Title: "First"},
		}, "sj-spec-auth-w1_aaa.md"),
		specEvent("auth-w1", []WaveStepDef{
			{ID: "s1", Title: "Duplicate"},
			{ID: "s2", Title: "Extra step"},
		}, "sj-spec-auth-w1_bbb.md"),
	}

	// when
	progress := ProjectWaveStepProgress(events)

	// then: first-wins, only 1 step
	targets := progress.PendingTargets()
	if len(targets) != 1 {
		t.Fatalf("expected 1 target (first-wins), got %d", len(targets))
	}
	if targets[0].Title != "First" {
		t.Errorf("title = %q, want First (first-wins)", targets[0].Title)
	}
}

func TestProjectWaveStepProgress_SingleStepWave(t *testing.T) {
	// given: spec with empty steps (single-step wave)
	events := []Event{
		specEvent("fix-w1", nil, "sj-spec-fix-w1.md"),
	}

	// when
	progress := ProjectWaveStepProgress(events)

	// then: implicit step with waveID as stepID
	targets := progress.PendingTargets()
	if len(targets) != 1 {
		t.Fatalf("expected 1 target for single-step wave, got %d", len(targets))
	}
	if targets[0].StepID != "fix-w1" {
		t.Errorf("stepID = %q, want fix-w1 (same as waveID)", targets[0].StepID)
	}
	if targets[0].ID != "fix-w1" {
		t.Errorf("ID = %q, want fix-w1 (single-step uses waveID only)", targets[0].ID)
	}
}

func TestProjectWaveStepProgress_HasWaves(t *testing.T) {
	// given: no events
	empty := ProjectWaveStepProgress(nil)
	if empty.HasWaves() {
		t.Error("empty progress should report HasWaves() = false")
	}

	// given: with spec
	events := []Event{
		specEvent("w1", []WaveStepDef{{ID: "s1", Title: "Step"}}, "spec.md"),
	}
	withSpec := ProjectWaveStepProgress(events)
	if !withSpec.HasWaves() {
		t.Error("progress with spec should report HasWaves() = true")
	}
}

// TestProjectWaveStepProgress_GoTaskboardScenario simulates the real go-taskboard
// state: 3 spec D-Mails with multiple steps. Expeditions complete some steps,
// skip others, and fail one. Verifies no duplicate target assignment.
func TestProjectWaveStepProgress_GoTaskboardScenario(t *testing.T) {
	// given: 3 specs (cluster-w1: 2 steps, cluster-w2: 2 steps, cluster-w3: 1 step)
	// Expedition results:
	//   cluster-w2:5 → success (PR #14)
	//   cluster-w1:1 → success (PR #15)
	//   cluster-w1:1 → skipped (already done by #15) — should NOT create dup target
	//   cluster-w2:5 → skipped (already done by #14) — should NOT create dup target
	//   cluster-w3:5 → success (PR #17)
	//   cluster-w2:1 → failed — should remain pending for retry
	events := []Event{
		specEvent("入力バリデーション:cluster-w1", []WaveStepDef{
			{ID: "2", Title: "DoD追記"},
			{ID: "3", Title: "DoD追記2"},
		}, "sj-spec-cluster-w1.md"),
		specEvent("入力バリデーション:cluster-w2", []WaveStepDef{
			{ID: "5", Title: "ErrInvalidStatus実装"},
			{ID: "1", Title: "handler層バリデーション"},
		}, "sj-spec-cluster-w2.md"),
		specEvent("入力バリデーション:cluster-w3", []WaveStepDef{
			{ID: "5", Title: "受け入れテスト追加"},
		}, "sj-spec-cluster-w3.md"),
		// Expedition #14: cluster-w2:5 success
		completedEvent("success", "入力バリデーション:cluster-w2", "5"),
		// Expedition #15: cluster-w1:1 — wait, this is step "1" which doesn't exist in w1
		// Let's use correct step IDs:
		// Expedition #15: cluster-w1:2 success
		completedEvent("success", "入力バリデーション:cluster-w1", "2"),
		// Expedition #16: cluster-w1:2 skipped (already done)
		completedEvent("skipped", "入力バリデーション:cluster-w1", "2"),
		// Expedition #17: cluster-w2:5 skipped (already done)
		completedEvent("skipped", "入力バリデーション:cluster-w2", "5"),
		// Expedition #18: cluster-w3:5 success
		completedEvent("success", "入力バリデーション:cluster-w3", "5"),
		// Expedition #19: cluster-w2:1 failed
		completedEvent("failed", "入力バリデーション:cluster-w2", "1"),
	}

	// when
	progress := ProjectWaveStepProgress(events)
	targets := progress.PendingTargets()

	// then: 2 pending targets
	// cluster-w1:3 (never attempted)
	// cluster-w2:1 (failed, retry-able)
	if len(targets) != 2 {
		t.Fatalf("expected 2 pending targets, got %d", len(targets))
	}
	if targets[0].WaveID != "入力バリデーション:cluster-w1" || targets[0].StepID != "3" {
		t.Errorf("targets[0] = %s:%s, want 入力バリデーション:cluster-w1:3", targets[0].WaveID, targets[0].StepID)
	}
	if targets[1].WaveID != "入力バリデーション:cluster-w2" || targets[1].StepID != "1" {
		t.Errorf("targets[1] = %s:%s, want 入力バリデーション:cluster-w2:1", targets[1].WaveID, targets[1].StepID)
	}
}

// TestProjectWaveStepProgress_NoDuplicateAfterSuccessAndSkip verifies the core fix:
// after a step is marked completed (success), subsequent skipped events for the same
// step do NOT create a new pending target.
func TestProjectWaveStepProgress_NoDuplicateAfterSuccessAndSkip(t *testing.T) {
	events := []Event{
		specEvent("w1", []WaveStepDef{
			{ID: "s1", Title: "Step 1"},
		}, "spec.md"),
		completedEvent("success", "w1", "s1"),
		completedEvent("skipped", "w1", "s1"),
		completedEvent("skipped", "w1", "s1"),
		completedEvent("skipped", "w1", "s1"),
	}

	progress := ProjectWaveStepProgress(events)
	targets := progress.PendingTargets()
	if len(targets) != 0 {
		t.Errorf("expected 0 targets after success + multiple skips, got %d", len(targets))
	}
}
