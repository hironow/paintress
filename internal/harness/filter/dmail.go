package filter

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
