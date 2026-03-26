//go:build scenario

package scenario_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// seedFailedJournals writes N journal files with the same failure reason to
// .expedition/journal/ inside the workspace repo. When N >= 2, Lumina scanning
// should produce a Defensive lumina containing the reason text.
//
// Journal format matches session.WriteJournal output:
//
//   - **Status**: failed
//   - **Reason**: {reason}
//   - **Insight**: {insight}  (optional, takes precedence over Reason)
func seedFailedJournals(t *testing.T, ws *Workspace, count int, reason, insight string) {
	t.Helper()
	journalDir := filepath.Join(ws.RepoPath, ".expedition", "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("create journal dir: %v", err)
	}

	for i := 1; i <= count; i++ {
		var insightLine string
		if insight != "" {
			insightLine = fmt.Sprintf("- **Insight**: %s\n", insight)
		}
		content := fmt.Sprintf(`# Expedition #%d — Journal
# This is a record of a past Expedition. Use the Insight field as a lesson for your mission.

- **Date**: 2026-03-20 00:00:00
- **Issue**: SEED-%03d — Pre-seeded failure
- **Mission**: implement
- **Status**: failed
- **Reason**: %s
- **PR**: none
- **Bugs found**: 0
- **Bug issues**: none
%s- **Failure type**: blocker
- **HIGH severity D-Mail**:
`, i, i, reason, insightLine)
		filename := fmt.Sprintf("%03d.md", i)
		if err := os.WriteFile(filepath.Join(journalDir, filename), []byte(content), 0o644); err != nil {
			t.Fatalf("write journal %s: %v", filename, err)
		}
	}
}

// seedSuccessJournals writes N journal files with the same mission/insight to
// .expedition/journal/. When N >= 3, Lumina scanning should produce an
// Offensive lumina containing the mission or insight text.
func seedSuccessJournals(t *testing.T, ws *Workspace, startIndex, count int, mission, insight string) {
	t.Helper()
	journalDir := filepath.Join(ws.RepoPath, ".expedition", "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("create journal dir: %v", err)
	}

	for i := startIndex; i < startIndex+count; i++ {
		var insightLine string
		if insight != "" {
			insightLine = fmt.Sprintf("- **Insight**: %s\n", insight)
		}
		content := fmt.Sprintf(`# Expedition #%d — Journal
# This is a record of a past Expedition. Use the Insight field as a lesson for your mission.

- **Date**: 2026-03-20 00:00:00
- **Issue**: SEED-%03d — Pre-seeded success
- **Mission**: %s
- **Status**: success
- **Reason**: completed
- **PR**: none
- **Bugs found**: 0
- **Bug issues**: none
%s- **Failure type**: none
- **HIGH severity D-Mail**:
`, i, i, mission, insightLine)
		filename := fmt.Sprintf("%03d.md", i)
		if err := os.WriteFile(filepath.Join(journalDir, filename), []byte(content), 0o644); err != nil {
			t.Fatalf("write journal %s: %v", filename, err)
		}
	}
}

// seedHighSeverityJournal writes a single journal with a HIGH severity D-Mail
// reference. Lumina scanning should produce an Alert lumina (threshold = 1).
func seedHighSeverityJournal(t *testing.T, ws *Workspace, index int, alertNames string) {
	t.Helper()
	journalDir := filepath.Join(ws.RepoPath, ".expedition", "journal")
	if err := os.MkdirAll(journalDir, 0o755); err != nil {
		t.Fatalf("create journal dir: %v", err)
	}

	content := fmt.Sprintf(`# Expedition #%d — Journal
# This is a record of a past Expedition. Use the Insight field as a lesson for your mission.

- **Date**: 2026-03-20 00:00:00
- **Issue**: SEED-%03d — Pre-seeded high severity
- **Mission**: implement
- **Status**: success
- **Reason**: completed
- **PR**: none
- **Bugs found**: 0
- **Bug issues**: none
- **Insight**:
- **Failure type**: none
- **HIGH severity D-Mail**: %s
`, index, index, alertNames)
	filename := fmt.Sprintf("%03d.md", index)
	if err := os.WriteFile(filepath.Join(journalDir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write journal %s: %v", filename, err)
	}
}

// TestScenario_LuminaDefensiveAppearsInPrompt verifies the full Lumina
// Defensive pipeline in a scenario test:
//
//  1. Pre-seed 2 failed journals with the same reason into .expedition/journal/
//  2. Inject a specification D-Mail into .expedition/inbox
//  3. Run paintress expedition (which calls ScanJournalsForLumina before worker start)
//  4. Read the prompt log written by fake-claude
//  5. Assert the prompt contains "Defensive (lessons from failures)" section header
//  6. Assert the prompt contains the failure reason text
//
// This confirms that past failure patterns are extracted and injected into the
// Claude prompt as defensive guidance.
func TestScenario_LuminaDefensiveAppearsInPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	// given — pre-seed 2 failed journals with the same reason (meets threshold >= 2)
	failureReason := "ruff lint violations in generated code"
	seedFailedJournals(t, ws, 2, failureReason, "")

	// Inject specification D-Mail into .expedition/inbox
	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-lumina-def-001",
		"kind":                 "specification",
		"description":          "Test specification for Lumina defensive verification",
	}, "# Test Spec\n\n## Actions\n\n- [add_dod] TEST-LUMINA-001: Verify Lumina defensive patterns")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-lumina-def-001.md", spec)

	// when — run paintress expedition
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("paintress expedition error (may be expected): %v", err)
	}

	// then — prompt log must contain Lumina Defensive section
	obs.AssertPromptCountAtLeast(1)
	obs.AssertPromptContainsLumina("Defensive (lessons from failures)")
	obs.AssertPromptContainsLumina(failureReason)
}

// TestScenario_LuminaDefensiveUsesInsightOverReason verifies that when
// journal entries have both Reason and Insight fields, the Insight text
// takes precedence in the Lumina Defensive pattern.
func TestScenario_LuminaDefensiveUsesInsightOverReason(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	// given — 2 journals with same insight (Insight takes precedence over Reason)
	insightText := "Redis connection pool exhausted under load"
	seedFailedJournals(t, ws, 2, "test timeout", insightText)

	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-lumina-insight-001",
		"kind":                 "specification",
		"description":          "Test specification for Lumina insight precedence",
	}, "# Test Spec\n\n## Actions\n\n- [add_dod] TEST-LUMINA-002: Verify Lumina insight precedence")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-lumina-insight-001.md", spec)

	// when
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("paintress expedition error (may be expected): %v", err)
	}

	// then — prompt should contain Insight text, NOT the raw Reason
	obs.AssertPromptCountAtLeast(1)
	obs.AssertPromptContainsLumina(insightText)
	obs.AssertPromptNotContainsLumina("[WARN] Avoid — failed 2 times: test timeout")
}

// TestScenario_LuminaOffensiveAppearsInPrompt verifies that success patterns
// (>= 3 occurrences) appear as Offensive luminas in the expedition prompt.
func TestScenario_LuminaOffensiveAppearsInPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	// given — 3 success journals with same insight (meets threshold >= 3)
	successInsight := "TDD cycle with parallel tests passes reliably"
	seedSuccessJournals(t, ws, 1, 3, "implement", successInsight)

	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-lumina-off-001",
		"kind":                 "specification",
		"description":          "Test specification for Lumina offensive verification",
	}, "# Test Spec\n\n## Actions\n\n- [add_dod] TEST-LUMINA-003: Verify Lumina offensive patterns")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-lumina-off-001.md", spec)

	// when
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("paintress expedition error (may be expected): %v", err)
	}

	// then
	obs.AssertPromptCountAtLeast(1)
	obs.AssertPromptContainsLumina("Offensive (proven patterns)")
	obs.AssertPromptContainsLumina(successInsight)
}

// TestScenario_LuminaAlertAppearsInPrompt verifies that HIGH severity D-Mail
// references from past journals appear as Alert luminas (threshold = 1).
func TestScenario_LuminaAlertAppearsInPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	// given — 1 journal with HIGH severity D-Mail (threshold = 1)
	seedHighSeverityJournal(t, ws, 1, "alert-critical-deploy")

	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-lumina-alert-001",
		"kind":                 "specification",
		"description":          "Test specification for Lumina alert verification",
	}, "# Test Spec\n\n## Actions\n\n- [add_dod] TEST-LUMINA-004: Verify Lumina alert patterns")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-lumina-alert-001.md", spec)

	// when
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("paintress expedition error (may be expected): %v", err)
	}

	// then
	obs.AssertPromptCountAtLeast(1)
	obs.AssertPromptContainsLumina("Alert (HIGH severity D-Mail from past expeditions)")
	obs.AssertPromptContainsLumina("alert-critical-deploy")
}

// TestScenario_LuminaNoneWhenNoJournals verifies that when no journal files
// exist, the prompt contains the "No Lumina learned" message instead of
// Defensive/Offensive sections.
func TestScenario_LuminaNoneWhenNoJournals(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	// given — no pre-seeded journals (fresh workspace)
	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-lumina-none-001",
		"kind":                 "specification",
		"description":          "Test specification for empty Lumina",
	}, "# Test Spec\n\n## Actions\n\n- [add_dod] TEST-LUMINA-005: Verify no Lumina message")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-lumina-none-001.md", spec)

	// when
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("paintress expedition error (may be expected): %v", err)
	}

	// then — prompt should contain "No Lumina learned" (from lumina_none message)
	obs.AssertPromptCountAtLeast(1)
	obs.AssertPromptContainsLumina("No Lumina learned")
	obs.AssertPromptNotContainsLumina("Defensive (lessons from failures)")
	obs.AssertPromptNotContainsLumina("Offensive (proven patterns)")
}

// TestScenario_LuminaBelowThresholdNotInPrompt verifies that failure patterns
// below the threshold (< 2) do NOT appear as Luminas, and success patterns
// below threshold (< 3) do NOT appear either.
func TestScenario_LuminaBelowThresholdNotInPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ws := NewWorkspace(t, "minimal")
	obs := NewObserver(ws, t)

	// given — 1 failure (below threshold of 2) + 2 successes (below threshold of 3)
	failReason := "unique-failure-for-threshold-test"
	seedFailedJournals(t, ws, 1, failReason, "")
	seedSuccessJournals(t, ws, 2, 2, "implement", "unique-success-for-threshold-test")

	spec := FormatDMail(map[string]string{
		"dmail-schema-version": "1",
		"name":                 "spec-lumina-threshold-001",
		"kind":                 "specification",
		"description":          "Test specification for Lumina threshold verification",
	}, "# Test Spec\n\n## Actions\n\n- [add_dod] TEST-LUMINA-006: Verify Lumina threshold")
	ws.InjectDMail(t, ".expedition", "inbox", "spec-lumina-threshold-001.md", spec)

	// when
	err := ws.RunPaintressExpedition(t, ctx)
	if err != nil {
		t.Logf("paintress expedition error (may be expected): %v", err)
	}

	// then — neither Defensive nor Offensive sections should appear
	obs.AssertPromptCountAtLeast(1)
	obs.AssertPromptNotContainsLumina("Defensive (lessons from failures)")
	obs.AssertPromptNotContainsLumina("Offensive (proven patterns)")
	obs.AssertPromptNotContainsLumina(failReason)
	obs.AssertPromptNotContainsLumina("unique-success-for-threshold-test")
}
