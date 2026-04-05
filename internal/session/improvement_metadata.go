package session

import "github.com/hironow/paintress/internal/domain"

func annotateReportDMail(mail *domain.DMail, report *domain.ExpeditionReport, expedition *Expedition) {
	if mail == nil {
		return
	}
	meta := correctionMetadataForReport(report, expedition)
	if meta.SchemaVersion == "" {
		return
	}
	mail.Metadata = meta.Apply(mail.Metadata)
}

func correctionMetadataForReport(report *domain.ExpeditionReport, expedition *Expedition) domain.CorrectionMetadata {
	if report == nil || expedition == nil {
		return domain.CorrectionMetadata{}
	}
	var candidates []domain.DMail
	candidates = append(candidates, expedition.InboxDMails...)
	candidates = append(candidates, expedition.MidMatchedDMails()...)
	for i := len(candidates) - 1; i >= 0; i-- {
		meta := domain.CorrectionMetadataFromMap(candidates[i].Metadata)
		if meta.SchemaVersion == "" || !matchesReportCorrection(candidates[i], report, expedition.Target) {
			continue
		}
		return meta.ForwardForRecheck()
	}
	return domain.CorrectionMetadata{}
}

func matchesReportCorrection(mail domain.DMail, report *domain.ExpeditionReport, target *domain.ExpeditionTarget) bool {
	if target != nil && mail.Wave != nil && mail.Wave.ID == target.WaveID {
		return mail.Wave.Step == "" || target.StepID == "" || mail.Wave.Step == target.StepID
	}
	if report.IssueID == "" {
		return false
	}
	for _, issueID := range mail.Issues {
		if issueID == report.IssueID {
			return true
		}
	}
	return false
}
