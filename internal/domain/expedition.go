package domain

import (
	"path/filepath"
)

// PromptData holds all dynamic values injected into the expedition prompt template.
type PromptData struct {
	Number          int
	Timestamp       string
	Bt              string // "`"
	Cb              string // "```"
	LuminaSection   string
	GradientSection string
	ReserveSection  string
	BaseBranch      string
	DevURL          string
	ContextSection  string
	InboxSection    string
	LinearTeam      string
	LinearProject   string
	MissionSection  string
}

// ContextDir returns the path to the context injection directory.
func ContextDir(continent string) string {
	return filepath.Join(continent, StateDir, "context")
}
