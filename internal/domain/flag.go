package domain

import "path/filepath"

// ExpeditionFlag represents the checkpoint on the Continent.
type ExpeditionFlag struct {
	LastExpedition  int
	LastUpdated     string
	LastIssue       string
	LastStatus      string
	Remaining       string
	CurrentIssue    string
	CurrentTitle    string
	MidHighSeverity int
}

func FlagPath(continent string) string {
	return filepath.Join(continent, ".expedition", ".run", "flag.md")
}

// BestFlag returns the flag with the highest LastExpedition number.
// If flags is empty, a zero-value ExpeditionFlag is returned.
func BestFlag(flags []ExpeditionFlag) ExpeditionFlag {
	if len(flags) == 0 {
		return ExpeditionFlag{}
	}
	best := flags[0]
	for _, f := range flags[1:] {
		if f.LastExpedition > best.LastExpedition {
			best = f
		}
	}
	return best
}
