package cmd

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// loadClaudeCmd returns the claude_cmd from project config, falling back to domain.DefaultClaudeCmd.
func loadClaudeCmd(repoPath string) string {
	cfg, err := session.LoadProjectConfig(repoPath)
	if err != nil {
		return domain.DefaultClaudeCmd
	}
	if cfg.ClaudeCmd != "" {
		return cfg.ClaudeCmd
	}
	return domain.DefaultClaudeCmd
}
