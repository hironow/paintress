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

	"go.opentelemetry.io/otel/attribute"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/platform"
)

// FetchIssuesViaMCP invokes Claude CLI with Linear MCP tools to fetch issues.
// Claude writes the result as a JSON array to a temp file in workDir.
func FetchIssuesViaMCP(ctx context.Context, claudeCmd, team, project, workDir string) ([]domain.Issue, error) {
	ctx, span := platform.Tracer.Start(ctx, "paintress.issues")
	defer span.End()

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

	start := time.Now()
	if err := cmd.Run(); err != nil {
		span.SetAttributes(attribute.Int64("issues.fetch.exec_ms", time.Since(start).Milliseconds()))
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "paintress.issues"))
		return nil, fmt.Errorf("claude: %w", err)
	}
	span.SetAttributes(attribute.Int64("issues.fetch.exec_ms", time.Since(start).Milliseconds()))

	data, err := os.ReadFile(outputPath)
	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "paintress.issues"))
		return nil, fmt.Errorf("read issues output: %w", err)
	}

	var issues []domain.Issue
	if err := json.Unmarshal(data, &issues); err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error.stage", "paintress.issues"))
		return nil, fmt.Errorf("parse issues output: %w", err)
	}

	span.SetAttributes(attribute.Int("issues.fetch.count", len(issues)))
	return issues, nil
}
