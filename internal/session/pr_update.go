package session

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// UpdatePRReviewGate appends or replaces the Review Gate section in a PR body
// using `gh pr edit`. The prURL must be a GitHub PR URL.
func UpdatePRReviewGate(ctx context.Context, prURL string, status domain.ReviewGateStatus, logger domain.Logger) error {
	section := status.FormatSection()

	// Extract PR number from URL (e.g. https://github.com/owner/repo/pull/42 → 42)
	prRef := extractPRRef(prURL)
	if prRef == "" {
		return fmt.Errorf("cannot extract PR reference from %q", prURL)
	}

	// Read current PR body
	viewCtx, viewCancel := context.WithTimeout(ctx, 30*time.Second)
	defer viewCancel()
	viewCmd := exec.CommandContext(viewCtx, "gh", "pr", "view", prRef, "--json", "body", "--jq", ".body")
	var viewOut bytes.Buffer
	viewCmd.Stdout = &viewOut
	if err := viewCmd.Run(); err != nil {
		return fmt.Errorf("gh pr view: %w", err)
	}
	currentBody := viewOut.String()

	// Build new body with review gate section
	newBody := domain.AppendReviewGateSection(currentBody, section)

	// Update PR body
	editCtx, editCancel := context.WithTimeout(ctx, 30*time.Second)
	defer editCancel()
	editCmd := exec.CommandContext(editCtx, "gh", "pr", "edit", prRef, "--body", newBody)
	if out, err := editCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh pr edit: %w\n%s", err, string(out))
	}

	logger.Info("PR %s: review gate status updated", prRef)
	return nil
}

// extractPRRef extracts a PR number or full URL usable with `gh pr` commands.
func extractPRRef(prURL string) string {
	// gh pr view accepts full URLs directly
	if strings.HasPrefix(prURL, "https://") {
		return prURL
	}
	return ""
}
