package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StatusReport holds operational status information for the paintress tool.
type StatusReport struct {
	Continent           string    `json:"continent"`
	Expeditions         int       `json:"expeditions"`
	Successes           int       `json:"successes"`
	Failures            int       `json:"failures"`
	SuccessRate         float64   `json:"success_rate"`
	GradientLevel       int       `json:"gradient_level"`
	InboxCount          int       `json:"inbox_count"`
	ArchiveCount        int       `json:"archive_count"`
	LastExpedition      time.Time `json:"last_expedition"`
	ProviderState       string    `json:"provider_state,omitempty"`
	ProviderReason      string    `json:"provider_reason,omitempty"`
	ProviderRetryBudget int       `json:"provider_retry_budget,omitempty"`
	ProviderResumeAt    time.Time `json:"provider_resume_at,omitempty"`
	ProviderResumeWhen  string    `json:"provider_resume_when,omitempty"`
}

// FormatText returns a human-readable status report string suitable for stdout.
func (r StatusReport) FormatText() string {
	var b strings.Builder
	b.WriteString("paintress status\n\n")

	fmt.Fprintf(&b, "  %-16s %s\n", "Continent:", r.Continent)

	// Expeditions with breakdown
	skipped := r.Expeditions - r.Successes - r.Failures
	fmt.Fprintf(&b, "  %-16s %d (%d success, %d failed, %d skipped)\n",
		"Expeditions:", r.Expeditions, r.Successes, r.Failures, skipped)

	// Success rate
	if r.Expeditions == 0 {
		fmt.Fprintf(&b, "  %-16s %s\n", "Success rate:", "no events")
	} else {
		fmt.Fprintf(&b, "  %-16s %.1f%%\n", "Success rate:", r.SuccessRate*100)
	}

	fmt.Fprintf(&b, "  %-16s level %d\n", "Gradient:", r.GradientLevel)
	fmt.Fprintf(&b, "  %-16s %d pending\n", "Inbox:", r.InboxCount)
	fmt.Fprintf(&b, "  %-16s %d processed\n", "Archive:", r.ArchiveCount)
	if r.ProviderState != "" {
		fmt.Fprintf(&b, "  %-16s %s", "Provider:", r.ProviderState)
		if r.ProviderReason != "" {
			fmt.Fprintf(&b, " (%s)", r.ProviderReason)
		}
		b.WriteByte('\n')
		if r.ProviderRetryBudget > 0 {
			fmt.Fprintf(&b, "  %-16s %d\n", "Retry budget:", r.ProviderRetryBudget)
		}
		if r.ProviderResumeWhen != "" {
			fmt.Fprintf(&b, "  %-16s %s\n", "Resume when:", r.ProviderResumeWhen)
		}
		if !r.ProviderResumeAt.IsZero() {
			fmt.Fprintf(&b, "  %-16s %s\n", "Resume at:", r.ProviderResumeAt.Format(time.RFC3339))
		}
	}

	// Last expedition
	if r.LastExpedition.IsZero() {
		fmt.Fprintf(&b, "  %-16s %s\n", "Last expedition:", "no expeditions yet")
	} else {
		fmt.Fprintf(&b, "  %-16s %s\n", "Last expedition:", r.LastExpedition.Format(time.RFC3339))
	}

	return b.String()
}

// FormatJSON returns the status report as a compact JSON string.
func (r StatusReport) FormatJSON() string {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}
