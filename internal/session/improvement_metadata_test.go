package session

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestCorrectionMetadataForReport_MatchesWaveTarget(t *testing.T) {
	meta := domain.CorrectionMetadata{
		FailureType:      domain.FailureTypeExecutionFailure,
		TargetAgent:      "paintress",
		CorrelationID:    "corr-wave",
		CorrectiveAction: "retry",
	}
	expedition := &Expedition{
		InboxDMails: []domain.DMail{{
			Name:     "feedback-1",
			Wave:     &domain.WaveReference{ID: "wave-1", Step: "step-1"},
			Metadata: meta.Apply(nil),
		}},
		Target: &domain.ExpeditionTarget{WaveID: "wave-1", StepID: "step-1"},
	}
	report := &domain.ExpeditionReport{IssueID: "ENG-1"}

	got := correctionMetadataForReport(report, expedition)

	if got.CorrelationID != "corr-wave" {
		t.Fatalf("CorrelationID = %q, want corr-wave", got.CorrelationID)
	}
	if got.TargetAgent != "" {
		t.Fatalf("TargetAgent = %q, want empty", got.TargetAgent)
	}
	if got.Outcome != domain.ImprovementOutcomePending {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, domain.ImprovementOutcomePending)
	}
}

func TestAnnotateReportDMail_UsesIssueMatchFallback(t *testing.T) {
	meta := domain.CorrectionMetadata{
		FailureType:      domain.FailureTypeExecutionFailure,
		TargetAgent:      "paintress",
		CorrelationID:    "corr-issue",
		CorrectiveAction: "retry",
	}
	expedition := &Expedition{
		InboxDMails: []domain.DMail{{
			Name:     "feedback-1",
			Issues:   []string{"ENG-2"},
			Metadata: meta.Apply(nil),
		}},
	}
	report := &domain.ExpeditionReport{IssueID: "ENG-2"}
	mail := domain.DMail{Name: "pt-report-eng-2", Kind: "report", Description: "done"}

	annotateReportDMail(&mail, report, expedition)

	got := domain.CorrectionMetadataFromMap(mail.Metadata)
	if got.CorrelationID != "corr-issue" {
		t.Fatalf("CorrelationID = %q, want corr-issue", got.CorrelationID)
	}
	if got.TargetAgent != "" {
		t.Fatalf("TargetAgent = %q, want empty", got.TargetAgent)
	}
}
