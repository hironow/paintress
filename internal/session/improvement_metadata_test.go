package session

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

// white-box-reason: session internals: tests unexported corrective metadata matching helper

func TestCorrectionMetadataForReport_MatchesWaveTarget(t *testing.T) {
	meta := domain.CorrectionMetadata{
		FailureType:      domain.FailureTypeExecutionFailure,
		Severity:         domain.SeverityMedium,
		TargetAgent:      "paintress",
		CorrelationID:    "corr-wave",
		CorrectiveAction: "retry",
		RetryAllowed:     domain.BoolPtr(true),
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
	if got.RetryAllowed == nil || !*got.RetryAllowed {
		t.Fatal("RetryAllowed = nil/false, want true")
	}
}

func TestAnnotateReportDMail_UsesIssueMatchFallback(t *testing.T) {
	meta := domain.CorrectionMetadata{
		FailureType:      domain.FailureTypeExecutionFailure,
		Severity:         domain.SeverityHigh,
		TargetAgent:      "paintress",
		CorrelationID:    "corr-issue",
		CorrectiveAction: "retry",
		RetryAllowed:     domain.BoolPtr(false),
		EscalationReason: "high-severity",
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
	if got.RetryAllowed == nil || *got.RetryAllowed {
		t.Fatal("RetryAllowed = nil/true, want false")
	}
	if got.EscalationReason != "high-severity" {
		t.Fatalf("EscalationReason = %q, want high-severity", got.EscalationReason)
	}
}

func TestCorrectionMetadataForReport_AcceptsLegacyV1WithoutSchemaVersion(t *testing.T) {
	expedition := &Expedition{
		InboxDMails: []domain.DMail{{
			Name:   "feedback-legacy",
			Issues: []string{"ENG-3"},
			Metadata: map[string]string{
				domain.MetadataFailureType:      string(domain.FailureTypeExecutionFailure),
				domain.MetadataSeverity:         "HIGH",
				domain.MetadataCorrelationID:    "corr-legacy",
				domain.MetadataCorrectiveAction: "retry",
			},
		}},
	}
	report := &domain.ExpeditionReport{IssueID: "ENG-3"}

	got := correctionMetadataForReport(report, expedition)

	if got.ConsumerSchemaVersion() != domain.ImprovementSchemaVersion {
		t.Fatalf("ConsumerSchemaVersion = %q, want %q", got.ConsumerSchemaVersion(), domain.ImprovementSchemaVersion)
	}
	if got.Severity != domain.SeverityHigh {
		t.Fatalf("Severity = %q, want %q", got.Severity, domain.SeverityHigh)
	}
	if got.Outcome != domain.ImprovementOutcomePending {
		t.Fatalf("Outcome = %q, want %q", got.Outcome, domain.ImprovementOutcomePending)
	}
}
