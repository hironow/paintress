package domain

import "fmt"

// TrackingMode determines the issue tracking backend.
// Wave mode (default) uses D-Mail archive as event source.
// Linear mode uses Linear MCP for issue tracking (legacy).
type TrackingMode string

const (
	// ModeWave is the default mode: waves and steps drive expedition targeting.
	// D-Mail archive/ is the single source of truth for wave state.
	ModeWave TrackingMode = "wave"

	// ModeLinear uses Linear MCP for issue tracking (existing behavior).
	ModeLinear TrackingMode = "linear"
)

// NewTrackingMode returns ModeLinear when linear is true, ModeWave otherwise.
func NewTrackingMode(linear bool) TrackingMode {
	if linear {
		return ModeLinear
	}
	return ModeWave
}

// IsLinear returns true when operating in Linear MCP mode.
func (m TrackingMode) IsLinear() bool { return m == ModeLinear }

// IsWave returns true when operating in Wave-centric mode.
func (m TrackingMode) IsWave() bool { return m == ModeWave }

// String returns the mode name.
func (m TrackingMode) String() string { return string(m) }

// RepoPath is an always-valid non-empty repository path.
type RepoPath struct{ v string } // nosemgrep: structure.multiple-exported-structs-go -- domain primitive family (RepoPath/Team/Project/Days) are cohesive value-object newtypes for validated inputs; splitting would fragment constructor+accessor pairs [permanent]

// NewRepoPath parses a raw string into a RepoPath.
// Returns an error if the path is empty.
func NewRepoPath(raw string) (RepoPath, error) {
	if raw == "" {
		return RepoPath{}, fmt.Errorf("RepoPath is required")
	}
	return RepoPath{v: raw}, nil
}

// String returns the underlying path string.
func (r RepoPath) String() string { return r.v }

// Team is a semantic wrapper for a Linear team key.
// Empty string is valid (means "not specified").
type Team struct{ v string } // nosemgrep: structure.multiple-exported-structs-go -- domain primitive family cohesive set; see RepoPath [permanent]

// NewTeam creates a Team from a raw string. All values are valid.
func NewTeam(raw string) Team { return Team{v: raw} }

// String returns the underlying team key.
func (t Team) String() string { return t.v }

// IsEmpty returns true when no team was specified.
func (t Team) IsEmpty() bool { return t.v == "" }

// Project is a semantic wrapper for a Linear project name.
// Empty string is valid (means "not specified").
type Project struct{ v string } // nosemgrep: structure.multiple-exported-structs-go -- domain primitive family cohesive set; see RepoPath [permanent]

// NewProject creates a Project from a raw string. All values are valid.
func NewProject(raw string) Project { return Project{v: raw} }

// String returns the underlying project name.
func (p Project) String() string { return p.v }

// IsEmpty returns true when no project was specified.
func (p Project) IsEmpty() bool { return p.v == "" }

// Days is an always-valid positive integer representing a day count.
type Days struct{ v int }

// NewDays parses a raw integer into a Days value.
// Returns an error if the value is not positive.
func NewDays(raw int) (Days, error) {
	if raw <= 0 {
		return Days{}, fmt.Errorf("Days must be positive")
	}
	return Days{v: raw}, nil
}

// Int returns the underlying day count.
func (d Days) Int() int { return d.v }
