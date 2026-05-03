package domain

import (
	"path/filepath"
)

// PromptData holds all dynamic values injected into the expedition prompt template.
//
// HasEventSourcedContract is the Rival Contract v1.1 (Phase 1.1A) gate that
// switches the expedition prompt's optional event-sourced domain glossary
// preamble. The caller computes it from the inbox D-Mail metadata (e.g. via
// harness/filter.HasEventSourcedContract) so that the renderer never needs
// the raw D-Mail slice. When false (or unset), the rendered prompt is
// byte-identical to the legacy v1 surface.
type PromptData struct { // nosemgrep: domain-primitives.public-string-field-go -- DevURL is a template injection field; newtype adds no safety benefit [permanent]
	Number                  int
	Timestamp               string
	Bt                      string // "`"
	Cb                      string // "```"
	LuminaSection           string
	GradientSection         string
	ReserveSection          string
	BaseBranch              string
	DevURL                  string
	ContextSection          string
	InboxSection            string
	LinearTeam              string
	LinearProject           string
	MissionSection          string
	WaveTarget              *ExpeditionTarget // non-nil in wave mode
	HasEventSourcedContract bool              // Rival Contract v1.1 glossary gate
}

// ContextDir returns the path to the context injection directory.
func ContextDir(continent string) string {
	return filepath.Join(continent, StateDir, "context")
}
