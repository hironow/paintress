package session

import (
	"os"
	"os/exec"
	"sort"
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

	// workDir is an internal worktree path, not user input.
	logCmd := exec.Command("git", "log", "--oneline", "-10") //nolint:gosec // internal path
	logCmd.Dir = workDir
	if out, err := logCmd.Output(); err == nil && len(out) > 0 {
		b.WriteString("Commits:\n")
		b.Write(out)
	}

	statCmd := exec.Command("git", "diff", "--stat") //nolint:gosec // internal path
	statCmd.Dir = workDir
	if out, err := statCmd.Output(); err == nil && len(out) > 0 {
		b.WriteString("\nUncommitted changes:\n")
		b.Write(out)
	}

	return b.String()
}

// resumeIncompleteExpeditions uses the CheckpointScanner port to find
// checkpoint events without subsequent completion, then validates that
// the referenced worktrees still exist on disk.
// Best-effort: returns nil if scanner is nil or returns no results.
func (p *Paintress) resumeIncompleteExpeditions() []IncompleteExpedition {
	if p.checkpointScanner == nil {
		return nil
	}
	candidates := p.checkpointScanner.FindIncompleteCheckpoints()
	if len(candidates) == 0 {
		return nil
	}

	var result []IncompleteExpedition
	for _, cp := range candidates {
		// Verify worktree still exists on disk
		if info, statErr := os.Stat(cp.WorkDir); statErr != nil || !info.IsDir() {
			p.Logger.Debug("checkpoint %d: worktree %s not found, skipping", cp.Expedition, cp.WorkDir)
			continue
		}
		result = append(result, IncompleteExpedition{
			Expedition: cp.Expedition,
			WorkDir:    cp.WorkDir,
			Phase:      CheckpointPhase(cp.Phase),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Expedition < result[j].Expedition
	})

	return result
}

// cleanOrphanWorktrees removes worktrees from previous sessions.
// Best-effort: errors are logged but do not block startup.
func (p *Paintress) cleanOrphanWorktrees() {
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
