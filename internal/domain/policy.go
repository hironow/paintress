package domain

// Policy represents an implicit reactive rule: WHEN [EVENT] THEN [COMMAND].
// See ADR S0014 for the POLICY pattern reference.
type Policy struct {
	Name    string    // unique identifier for the policy
	Trigger EventType // domain event that activates this policy
	Action  string    // description of the resulting command
}

// Policies registers all known implicit policies in paintress.
// These document the existing reactive behaviors for future automation.
var Policies = []Policy{
	{Name: "ExpeditionCompletedStageReport", Trigger: EventExpeditionCompleted, Action: "StageReport"},
	{Name: "InboxReceivedProcessFeedback", Trigger: EventInboxReceived, Action: "ProcessFeedback"},
	{Name: "GradientChangedTriggerGommage", Trigger: EventGradientChanged, Action: "TriggerGommage"},
	{Name: "DMailStagedFlushOutbox", Trigger: EventDMailStaged, Action: "FlushOutbox"},
}
