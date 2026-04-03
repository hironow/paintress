package policy

import (
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// FormatDMailForPrompt formats d-mails as a human-readable Markdown section
// for injection into expedition prompts. Returns empty string for empty input.
func FormatDMailForPrompt(dmails []domain.DMail) string {
	if len(dmails) == 0 {
		return ""
	}
	var buf strings.Builder
	for _, dm := range dmails {
		fmt.Fprintf(&buf, "### %s (%s)\n\n", dm.Name, dm.Kind)
		fmt.Fprintf(&buf, "**Description:** %s\n", dm.Description)
		if len(dm.Issues) > 0 {
			fmt.Fprintf(&buf, "**Issues:** %s\n", strings.Join(dm.Issues, ", "))
		}
		if dm.Severity != "" {
			fmt.Fprintf(&buf, "**Severity:** %s\n", dm.Severity)
		}
		if dm.Body != "" {
			buf.WriteString("\n")
			buf.WriteString(dm.Body)
			if !strings.HasSuffix(dm.Body, "\n") {
				buf.WriteString("\n")
			}
		}
		buf.WriteString("\n")
	}
	return buf.String()
}

// NewReportDMail creates a report d-mail from an ExpeditionReport.
// gaugeLevel is the current GradientGauge level and determines the Severity field.
func NewReportDMail(report *domain.ExpeditionReport, gaugeLevel int) domain.DMail {
	name := "pt-report-" + domain.SanitizeDMailKey(report.IssueID) + "_" + domain.DMailUUIDFunc()

	var body strings.Builder
	fmt.Fprintf(&body, "# Expedition #%d Report: %s\n\n", report.Expedition, report.IssueTitle)
	fmt.Fprintf(&body, "- **Issue:** %s\n", report.IssueID)
	fmt.Fprintf(&body, "- **Mission:** %s\n", report.MissionType)
	fmt.Fprintf(&body, "- **Status:** %s\n", report.Status)
	if report.PRUrl != "" && report.PRUrl != "none" {
		fmt.Fprintf(&body, "- **PR:** %s\n", report.PRUrl)
	}
	if report.Reason != "" {
		fmt.Fprintf(&body, "\n## Summary\n\n%s\n", report.Reason)
	}

	dm := domain.DMail{
		Name:          name,
		Kind:          "report",
		Description:   fmt.Sprintf("Expedition #%d completed %s for %s", report.Expedition, report.MissionType, report.IssueID),
		Issues:        []string{report.IssueID},
		Severity:      ReportSeverity(gaugeLevel),
		SchemaVersion: domain.DMailSchemaVersion,
		Body:          body.String(),
	}

	if report.Insight != "" {
		dm.Context = &domain.InsightContext{
			Insights: []domain.InsightSummary{
				{Source: report.IssueID, Summary: report.Insight},
			},
		}
	}

	// Wave-centric mode: attach wave reference for archive projection
	if report.WaveID != "" {
		dm.Wave = &domain.WaveReference{
			ID:   report.WaveID,
			Step: report.StepID,
		}
		// Override name to include wave/step for uniqueness
		if report.StepID != "" {
			dm.Name = "pt-report-" + domain.SanitizeDMailKey(report.WaveID+"-"+report.StepID) + "_" + domain.DMailUUIDFunc()
		} else {
			dm.Name = "pt-report-" + domain.SanitizeDMailKey(report.WaveID) + "_" + domain.DMailUUIDFunc()
		}
	}

	return dm
}

// BuildFollowUpPrompt builds a follow-up prompt for issue-matched D-Mails
// received mid-expedition. Returns empty string for empty input.
func BuildFollowUpPrompt(dmails []domain.DMail) string {
	if len(dmails) == 0 {
		return ""
	}
	var buf strings.Builder
	buf.WriteString("The following D-Mail(s) arrived during this expedition and are related to the issue you just worked on.\n")
	buf.WriteString("Review them and take any additional action if needed. If no action is required, briefly acknowledge.\n\n")
	buf.WriteString(FormatDMailForPrompt(dmails))
	return buf.String()
}

// FilterHighSeverity returns only HIGH severity d-mails from the input slice.
func FilterHighSeverity(dmails []domain.DMail) []domain.DMail {
	var high []domain.DMail
	for _, dm := range dmails {
		if dm.Severity == "high" {
			high = append(high, dm)
		}
	}
	return high
}
