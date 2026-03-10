package session

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/hironow/paintress/internal/platform"
)

// PreflightCheck verifies that required binaries are available in PATH.
// Unlike a full doctor check, this uses platform.LookPathShell to handle
// shell-like command strings (e.g. "KEY=VAL cmd arg").
func PreflightCheck(binaries ...string) error {
	for _, bin := range binaries {
		if _, err := platform.LookPathShell(bin); err != nil {
			_, resolved, _ := platform.ParseShellCommand(bin)
			return fmt.Errorf("preflight: %s not found in PATH (from %q)", resolved, bin)
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
