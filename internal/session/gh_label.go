package session

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// PROpenLabel is the label applied to issues when paintress creates a PR.
const PROpenLabel = "paintress:pr-open"

// ApplyIssueLabel creates a label (idempotent) and applies it to a GitHub issue.
func ApplyIssueLabel(ctx context.Context, issueID, label string, logger domain.Logger) error {
	if issueID == "" {
		return fmt.Errorf("apply label: issue ID is empty")
	}
	if label == "" {
		return fmt.Errorf("apply label: label is empty")
	}

	// Ensure label exists (--force is idempotent)
	createCtx, createCancel := context.WithTimeout(ctx, 15*time.Second)
	defer createCancel()
	createCmd := exec.CommandContext(createCtx, "gh", "label", "create", label, "--force")
	if out, err := createCmd.CombinedOutput(); err != nil {
		if logger != nil {
			logger.Warn("gh label create %s: %v (%s)", label, err, string(out))
		}
		// Non-fatal: label may already exist
	}

	// Apply to issue
	editCtx, editCancel := context.WithTimeout(ctx, 15*time.Second)
	defer editCancel()
	editCmd := exec.CommandContext(editCtx, "gh", "issue", "edit", issueID, "--add-label", label)
	if out, err := editCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh issue edit --add-label %s: %w (%s)", label, err, string(out))
	}

	if logger != nil {
		logger.Info("Issue %s: label %s applied", issueID, label)
	}
	return nil
}
