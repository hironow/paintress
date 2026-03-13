package session

import (
	"fmt"
	"strings"

	"github.com/hironow/paintress/internal/domain"
)

// checkClaudeAuth determines if the Claude CLI is authenticated by
// interpreting the result of running `claude mcp list`. A successful
// command execution (no error) indicates the CLI is authenticated.
// claudeCmd is the configured command string (may include env prefix).
func checkClaudeAuth(mcpOutput string, mcpErr error, claudeCmd string) domain.DoctorCheck {
	if mcpErr != nil {
		hint := buildLoginHint(claudeCmd)
		return domain.DoctorCheck{
			Name:    "claude-auth",
			Status:  domain.CheckWarn,
			Message: "not authenticated: " + mcpErr.Error(),
			Hint:    hint,
		}
	}
	return domain.DoctorCheck{
		Name:    "claude-auth",
		Status:  domain.CheckOK,
		Message: "authenticated",
	}
}

// buildLoginHint constructs a login hint that preserves any env prefix
// from the configured claude command (e.g. "CLAUDE_CONFIG_DIR=/path claude").
func buildLoginHint(claudeCmd string) string {
	envPrefix := extractEnvPrefix(claudeCmd)
	if envPrefix == "" {
		return `run "claude login" to authenticate`
	}
	return fmt.Sprintf(`run "%s claude login" to authenticate`, envPrefix)
}

// extractEnvPrefix extracts leading KEY=VALUE pairs from a command string.
// Returns the env prefix portion or empty string if none.
func extractEnvPrefix(cmd string) string {
	parts := strings.Fields(cmd)
	var envParts []string
	for _, p := range parts {
		if strings.Contains(p, "=") {
			envParts = append(envParts, p)
		} else {
			break
		}
	}
	return strings.Join(envParts, " ")
}

// checkLinearMCP parses `claude mcp list` output for Linear MCP connection.
// Looks for a line containing "linear", "✓", and "connected" (case-insensitive).
// Requires "✓" to avoid false positives from "disconnected" or "not connected".
func checkLinearMCP(mcpOutput string, mcpErr error) domain.DoctorCheck {
	if mcpErr != nil {
		return domain.DoctorCheck{
			Name:    "linear-mcp",
			Status:  domain.CheckWarn,
			Message: fmt.Sprintf("claude mcp list failed: %v", mcpErr),
			Hint:    `run "claude login" to authenticate`,
		}
	}
	output := strings.ToLower(mcpOutput)
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "linear") &&
			strings.Contains(line, "✓") &&
			strings.Contains(line, "connected") {
			return domain.DoctorCheck{
				Name:    "linear-mcp",
				Status:  domain.CheckOK,
				Message: "Linear MCP connected",
			}
		}
	}
	return domain.DoctorCheck{
		Name:    "linear-mcp",
		Status:  domain.CheckWarn,
		Message: "Linear MCP not found or not connected",
		Hint: "run \"claude mcp add --transport http --scope project linear https://mcp.linear.app/mcp\" in your project root\n" +
			"  (a fully compatible local-only Linear MCP alternative is planned — check the project README for updates)",
	}
}

// checkClaudeInference determines if the Claude CLI can perform inference
// by interpreting the result of a minimal "1+1=" prompt.
func checkClaudeInference(output string, err error) domain.DoctorCheck {
	if err != nil {
		return domain.DoctorCheck{
			Name:    "claude-inference",
			Status:  domain.CheckWarn,
			Message: "inference failed: " + err.Error(),
			Hint: `"signal: killed" = CLI startup too slow (timeout 3m); ` +
				`"nested session" = CLAUDECODE env var leaked (doctor should filter it); ` +
				`otherwise check API key, quota, and model access`,
		}
	}
	if strings.TrimSpace(output) != "2" {
		return domain.DoctorCheck{
			Name:    "claude-inference",
			Status:  domain.CheckWarn,
			Message: "unexpected response: " + strings.TrimSpace(output),
			Hint:    "model returned unexpected output; check model access and API quota",
		}
	}
	return domain.DoctorCheck{
		Name:    "claude-inference",
		Status:  domain.CheckOK,
		Message: "inference OK",
	}
}

// requiredGHScopes lists OAuth scopes that paintress needs for full
// functionality (e.g. gh pr edit requires read:project when PRs are linked
// to GitHub Projects).
var requiredGHScopes = []string{"repo", "read:project"}

// checkGHScopes verifies that the gh CLI token has the required OAuth scopes.
// Parses the output of `gh auth status` for the "Token scopes:" line.
func checkGHScopes(output string, err error) domain.DoctorCheck {
	if err != nil {
		return domain.DoctorCheck{
			Name:    "gh-scopes",
			Status:  domain.CheckWarn,
			Message: "not authenticated: " + err.Error(),
			Hint:    `run "gh auth login" to authenticate`,
		}
	}

	// Find "Token scopes:" line
	var scopesLine string
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "Token scopes:") {
			scopesLine = line
			break
		}
	}
	if scopesLine == "" {
		return domain.DoctorCheck{
			Name:    "gh-scopes",
			Status:  domain.CheckWarn,
			Message: "could not determine token scopes",
			Hint:    `run "gh auth status" to check your token`,
		}
	}

	var missing []string
	for _, scope := range requiredGHScopes {
		if !strings.Contains(scopesLine, scope) {
			missing = append(missing, scope)
		}
	}

	if len(missing) > 0 {
		return domain.DoctorCheck{
			Name:    "gh-scopes",
			Status:  domain.CheckWarn,
			Message: "missing scopes: " + strings.Join(missing, ", "),
			Hint:    fmt.Sprintf(`run "gh auth refresh -s %s" to add missing scopes`, strings.Join(missing, " -s ")),
		}
	}

	return domain.DoctorCheck{
		Name:    "gh-scopes",
		Status:  domain.CheckOK,
		Message: "scopes OK",
	}
}
