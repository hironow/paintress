package session

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hironow/paintress/internal/domain"
)

// FetchIssuesViaMCP invokes Claude CLI with Linear MCP tools to fetch issues.
// Claude writes the result as a JSON array to a temp file in workDir.
func FetchIssuesViaMCP(ctx context.Context, claudeCmd, team, project, workDir string) ([]domain.Issue, error) {
	outputPath := filepath.Join(workDir, fmt.Sprintf("issues-%d.json", time.Now().UnixNano()))

	var projectClause string
	if project != "" {
		projectClause = fmt.Sprintf(" for project %q", project)
	}
	prompt := fmt.Sprintf(
		"Use mcp__linear__list_issues to list ALL issues for team %q%s. "+
			"Paginate until no more results. "+
			"Write the result as a JSON array to %s "+
			"Each element must have fields: id (the issue identifier like TEAM-123), title, priority (number), status (state name), labels (array of label names).",
		team, projectClause, outputPath,
	)

	args := []string{
		"--print",
		"--dangerously-skip-permissions",
		"--allowedTools", "mcp__linear__list_issues,Write",
		"-p", prompt,
	}

	cmd := exec.CommandContext(ctx, claudeCmd, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude: %w", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("read issues output: %w", err)
	}

	var issues []domain.Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, fmt.Errorf("parse issues output: %w", err)
	}

	return issues, nil
}
