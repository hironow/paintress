package domain

import "fmt"

// NewEscalationDMail creates a feedback D-Mail for escalation when
// consecutive expedition failures reach the threshold. The D-Mail is
// HIGH severity and targets phonewave delivery via the outbox.
func NewEscalationDMail(expedition, failureCount int) DMail {
	return DMail{
		Name:          fmt.Sprintf("feedback-escalation-exp%d", expedition),
		Kind:          "feedback",
		Description:   fmt.Sprintf("Escalation: %d consecutive expedition failures at expedition #%d", failureCount, expedition),
		Severity:      "high",
		SchemaVersion: DMailSchemaVersion,
		Body: fmt.Sprintf("# Escalation Report\n\n"+
			"Expedition #%d triggered escalation after %d consecutive failures.\n\n"+
			"## Recommended Actions\n\n"+
			"- Review journal entries for failure details\n"+
			"- Check expedition context and issue configuration\n"+
			"- Investigate root cause before re-running\n",
			expedition, failureCount),
	}
}
