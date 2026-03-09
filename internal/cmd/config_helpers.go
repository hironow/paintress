package cmd

import (
	"github.com/hironow/paintress/internal/session"
)

// loadClaudeCmd returns the claude_cmd from project config, falling back to "claude".
func loadClaudeCmd(repoPath string) string {
	cfg, err := session.LoadProjectConfig(repoPath)
	if err != nil {
		return "claude"
	}
	if cfg.ClaudeCmd != "" {
		return cfg.ClaudeCmd
	}
	return "claude"
}
