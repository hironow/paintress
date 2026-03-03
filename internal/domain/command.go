package domain

import "fmt"

// RunExpeditionCommand represents the intent to run a paintress expedition.
// Independent of cobra — framework concerns are separated at the cmd layer.
type RunExpeditionCommand struct {
	RepoPath string
}

// Validate checks that the command has valid required fields.
func (c *RunExpeditionCommand) Validate() []error {
	var errs []error
	if c.RepoPath == "" {
		errs = append(errs, fmt.Errorf("RepoPath is required"))
	}
	return errs
}

// InitCommand represents the intent to initialize a paintress project.
type InitCommand struct {
	RepoPath string
	Team     string
	Project  string
}

// Validate checks that the command has valid required fields.
func (c *InitCommand) Validate() []error {
	var errs []error
	if c.RepoPath == "" {
		errs = append(errs, fmt.Errorf("RepoPath is required"))
	}
	return errs
}

// ArchivePruneCommand represents the intent to prune old archive files.
type ArchivePruneCommand struct {
	RepoPath string
	Days     int
	Execute  bool
}

// Validate checks that the command has valid required fields.
func (c *ArchivePruneCommand) Validate() []error {
	var errs []error
	if c.RepoPath == "" {
		errs = append(errs, fmt.Errorf("RepoPath is required"))
	}
	if c.Days <= 0 {
		errs = append(errs, fmt.Errorf("Days must be positive"))
	}
	return errs
}
