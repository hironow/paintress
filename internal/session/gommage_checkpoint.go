package session

import (
	"os/exec"
	"strings"
	"time"
)

// CheckpointPhase tracks expedition progress for resume.
type CheckpointPhase string

const (
	CheckpointWorktreeReady   CheckpointPhase = "worktree_ready"
	CheckpointSubprocessStart CheckpointPhase = "subprocess_started"
)

// IncompleteExpedition represents an unfinished expedition found at startup.
type IncompleteExpedition struct {
	Expedition int
	WorkDir    string
	Phase      CheckpointPhase
}

// saveCheckpoint records expedition progress as an event.
func (p *Paintress) saveCheckpoint(exp int, phase CheckpointPhase, workDir string) {
	commitCount := countCommitsInDir(workDir)
	_ = p.Emitter.EmitCheckpoint(exp, string(phase), workDir, commitCount, time.Now())
}

// countCommitsInDir returns the number of commits on HEAD.
// Best-effort: returns 0 on any error.
func countCommitsInDir(workDir string) int {
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	// workDir is an internal worktree path, not user input.
	cmd := exec.Command("git", "rev-list", "--count", "HEAD") //nolint:gosec // internal path
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(out))
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// buildResumeContext generates lightweight context for --continue.
// Uses git log --oneline + git diff --stat (Claude reads files via Read tool).
func buildResumeContext(workDir string) string {
	var b strings.Builder
	b.WriteString("Previous progress in worktree:\n")

	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	// workDir is an internal worktree path, not user input.
	logCmd := exec.Command("git", "log", "--oneline", "-10") //nolint:gosec // internal path
	logCmd.Dir = workDir
	if out, err := logCmd.Output(); err == nil && len(out) > 0 {
		b.WriteString("Commits:\n")
		b.Write(out)
	}

	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	statCmd := exec.Command("git", "diff", "--stat") //nolint:gosec // internal path
	statCmd.Dir = workDir
	if out, err := statCmd.Output(); err == nil && len(out) > 0 {
		b.WriteString("\nUncommitted changes:\n")
		b.Write(out)
	}

	return b.String()
}

// cleanOrphanWorktrees removes worktrees from previous sessions.
// Best-effort: errors are logged but do not block startup.
func (p *Paintress) cleanOrphanWorktrees() {
	// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
	cmd := exec.Command("git", "worktree", "list", "--porcelain") //nolint:gosec // internal path
	cmd.Dir = p.config.Continent
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")
			if strings.Contains(path, "paintress-wt-") {
				// nosemgrep: go.lang.security.audit.dangerous-exec-command.dangerous-exec-command
				rmCmd := exec.Command("git", "worktree", "remove", "--force", path) //nolint:gosec // internal path
				rmCmd.Dir = p.config.Continent
				if rmErr := rmCmd.Run(); rmErr != nil {
					p.Logger.Warn("orphan worktree cleanup: %v", rmErr)
				} else {
					p.Logger.Info("cleaned orphan worktree: %s", path)
				}
			}
		}
	}
}
