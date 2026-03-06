package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// StatusReport holds operational status information for the paintress tool.
type StatusReport struct {
	Continent      string    `json:"continent"`
	Expeditions    int       `json:"expeditions"`
	Successes      int       `json:"successes"`
	Failures       int       `json:"failures"`
	SuccessRate    float64   `json:"success_rate"`
	GradientLevel  int       `json:"gradient_level"`
	InboxCount     int       `json:"inbox_count"`
	ArchiveCount   int       `json:"archive_count"`
	LastExpedition time.Time `json:"last_expedition"`
}

// FormatText returns a human-readable status report string suitable for stderr.
func (r StatusReport) FormatText() string {
	var b strings.Builder
	b.WriteString("paintress status:\n")

	// Continent
	b.WriteString(fmt.Sprintf("  Continent:       %s\n", r.Continent))

	// Expeditions with breakdown
	skipped := r.Expeditions - r.Successes - r.Failures
	b.WriteString(fmt.Sprintf("  Expeditions:     %d (%d success, %d failed, %d skipped)\n",
		r.Expeditions, r.Successes, r.Failures, skipped))

	// Success rate
	if r.Expeditions == 0 {
		b.WriteString("  Success rate:    no events\n")
	} else {
		b.WriteString(fmt.Sprintf("  Success rate:    %.1f%%\n", r.SuccessRate*100))
	}

	// Gradient
	b.WriteString(fmt.Sprintf("  Gradient:        level %d\n", r.GradientLevel))

	// Inbox
	b.WriteString(fmt.Sprintf("  Inbox:           %d pending\n", r.InboxCount))

	// Archive
	b.WriteString(fmt.Sprintf("  Archive:         %d processed\n", r.ArchiveCount))

	// Last expedition
	if r.LastExpedition.IsZero() {
		b.WriteString("  Last expedition: no expeditions yet\n")
	} else {
		b.WriteString(fmt.Sprintf("  Last expedition: %s\n", r.LastExpedition.Format(time.RFC3339)))
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
