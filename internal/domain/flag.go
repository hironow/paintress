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
