package filter

import (
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// FormatLuminaForPrompt formats Luminas for injection into the expedition prompt.
func FormatLuminaForPrompt(luminas []domain.Lumina) string {
	if len(luminas) == 0 {
		return domain.Msg("lumina_none")
	}

	var alerts, defensive, offensive []string
	for _, l := range luminas {
		switch l.Source {
		case "high-severity-alert":
			alerts = append(alerts, fmt.Sprintf("- %s", l.Pattern))
		case "failure-pattern":
			defensive = append(defensive, fmt.Sprintf("- %s", l.Pattern))
		case "success-pattern":
			offensive = append(offensive, fmt.Sprintf("- %s", l.Pattern))
		}
	}

	var sb strings.Builder
	if len(alerts) > 0 {
		sb.WriteString("## Alert (HIGH severity D-Mail from past expeditions)\n")
		for _, a := range alerts {
			sb.WriteString(a)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	if len(defensive) > 0 {
		sb.WriteString(domain.Msg("lumina_defensive"))
		sb.WriteString("\n")
		for _, d := range defensive {
			sb.WriteString(d)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	if len(offensive) > 0 {
		sb.WriteString(domain.Msg("lumina_offensive"))
		sb.WriteString("\n")
		for _, o := range offensive {
			sb.WriteString(o)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
