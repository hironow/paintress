package session

import (
	"fmt"
	"os/exec"
	"strings"
)

// PreflightCheck verifies that required binaries are available in PATH.
// Unlike a full doctor check, this only uses exec.LookPath (no version execution).
func PreflightCheck(binaries ...string) error {
	for _, bin := range binaries {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("preflight: %s not found in PATH", bin)
		}
	}
	return nil
}

// PreflightCheckRemote verifies that the git repository at repoDir has at
// least one remote configured. Paintress creates GitHub Pull Requests for
// Linear issues, so a remote is required for `git push` and PR creation.
func PreflightCheckRemote(repoDir string) error {
	cmd := exec.Command("git", "remote")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("preflight: failed to check git remote in %s: %w", repoDir, err)
	}
	if strings.TrimSpace(string(out)) == "" {
		return fmt.Errorf("preflight: no git remote configured in %s — "+
			"paintress creates Pull Requests for Linear issues, so a remote repository is required. "+
			"Run 'git remote add origin <url>' to connect to GitHub", repoDir)
	}
	return nil
}
