// white-box-reason: tests unexported loadClaudeCmd function
package cmd

import (
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestLoadClaudeCmd_Default(t *testing.T) {
	// given — no config file
	dir := t.TempDir()

	// when
	cmd := loadClaudeCmd(dir)

	// then — should return default
	if cmd != "claude" {
		t.Errorf("loadClaudeCmd default = %q, want %q", cmd, "claude")
	}
}

func TestLoadClaudeCmd_FromConfig(t *testing.T) {
	// given — config with custom claude_cmd
	dir := t.TempDir()
	cfg := &domain.ProjectConfig{ClaudeCmd: "custom-claude"}
	if err := session.SaveProjectConfig(dir, cfg); err != nil {
		t.Fatalf("SaveProjectConfig: %v", err)
	}

	// when
	cmd := loadClaudeCmd(dir)

	// then
	if cmd != "custom-claude" {
		t.Errorf("loadClaudeCmd = %q, want %q", cmd, "custom-claude")
	}
}
