package paintress

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
