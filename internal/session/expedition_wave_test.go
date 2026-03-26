package session

// white-box-reason: tests wave-mode prompt template branch using unexported Expedition struct

import (
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestBuildPrompt_WaveTargetSection(t *testing.T) {
	// given: expedition with wave target
	e := newTestExpedition(t, "", 0)
	e.Target = &domain.ExpeditionTarget{
		ID:          "auth-w1:s1",
		WaveID:      "auth-w1",
		StepID:      "s1",
		Title:       "Add JWT middleware",
		Description: "Authentication Foundation",
		Acceptance:  "Middleware intercepts all /api/* routes",
	}

	// when
	prompt := e.BuildPrompt()

	// then: wave target section present
	if !strings.Contains(prompt, "Wave Target") && !strings.Contains(prompt, "Wave ターゲット") {
		t.Error("expected Wave Target section in prompt")
	}
	if !strings.Contains(prompt, "auth-w1:s1") {
		t.Error("expected step ID in prompt")
	}
	if !strings.Contains(prompt, "Add JWT middleware") {
		t.Error("expected step title in prompt")
	}
	if !strings.Contains(prompt, "Middleware intercepts all /api/* routes") {
		t.Error("expected acceptance criteria in prompt")
	}
	// Linear scope should NOT be present
	if strings.Contains(prompt, "Linear Scope") || strings.Contains(prompt, "Linear スコープ") {
		t.Error("wave mode should not include Linear Scope section")
	}
}

func TestBuildPrompt_NoWaveTarget_NoWaveSection(t *testing.T) {
	// given: expedition without wave target (linear mode)
	e := newTestExpedition(t, "", 0)

	// when
	prompt := e.BuildPrompt()

	// then: no Wave Target section
	if strings.Contains(prompt, "Wave Target") || strings.Contains(prompt, "Wave ターゲット") {
		t.Error("linear mode should not include Wave Target section")
	}
}

func TestNewReportDMail_WaveReference(t *testing.T) {
	// given: expedition report with wave context
	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "auth-w1:s1",
		IssueTitle:  "Add JWT middleware",
		MissionType: "implement",
		Status:      "success",
		WaveID:      "auth-w1",
		StepID:      "s1",
	}

	// when
	dm := domain.NewReportDMail(report, 3)

	// then
	if dm.Wave == nil {
		t.Fatal("expected wave reference in report D-Mail")
	}
	if dm.Wave.ID != "auth-w1" {
		t.Errorf("wave.id = %q, want auth-w1", dm.Wave.ID)
	}
	if dm.Wave.Step != "s1" {
		t.Errorf("wave.step = %q, want s1", dm.Wave.Step)
	}
	if dm.Name != "report-auth-w1-s1" {
		t.Errorf("name = %q, want report-auth-w1-s1", dm.Name)
	}
}

func TestNewReportDMail_NoWaveReference_LinearMode(t *testing.T) {
	// given: report without wave context (linear mode)
	report := &domain.ExpeditionReport{
		Expedition:  1,
		IssueID:     "MY-42",
		IssueTitle:  "Fix bug",
		MissionType: "fix",
		Status:      "success",
	}

	// when
	dm := domain.NewReportDMail(report, 2)

	// then: no wave reference
	if dm.Wave != nil {
		t.Error("expected no wave reference in linear mode report")
	}
	if !strings.HasPrefix(dm.Name, "report-my-42") {
		t.Errorf("name = %q, want report-my-42 prefix", dm.Name)
	}
}
