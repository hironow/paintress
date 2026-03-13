package session_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func TestCheckClaudeAuth_Failed_HasHint(t *testing.T) {
	// given
	check := session.ExportCheckClaudeAuth("", fmt.Errorf("exit status 1"), "claude")

	// when/then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if check.Hint == "" {
		t.Error("expected hint for failed auth")
	}
	if !strings.Contains(check.Hint, "claude login") {
		t.Errorf("hint should mention 'claude login', got: %s", check.Hint)
	}
}

func TestCheckClaudeAuth_WithEnvPrefix_HasHint(t *testing.T) {
	// given: env-prefixed command
	check := session.ExportCheckClaudeAuth("", fmt.Errorf("exit status 1"), "CLAUDE_CONFIG_DIR=/foo claude")

	// when/then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if !strings.Contains(check.Hint, "CLAUDE_CONFIG_DIR=/foo") {
		t.Errorf("hint should include env prefix, got: %s", check.Hint)
	}
	if !strings.Contains(check.Hint, "login") {
		t.Errorf("hint should mention login, got: %s", check.Hint)
	}
}

func TestCheckLinearMCP_NotConnected_HasHint(t *testing.T) {
	// given
	mcpOutput := "plugin:filesystem:filesystem: /path (stdio) - ok Connected\n"

	// when
	check := session.ExportCheckLinearMCP(mcpOutput, nil)

	// then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if check.Hint == "" {
		t.Error("expected hint for disconnected linear MCP")
	}
	if !strings.Contains(check.Hint, "claude mcp add") {
		t.Errorf("hint should mention 'claude mcp add', got: %s", check.Hint)
	}
}

func TestCheckContinent_Missing_HasHint(t *testing.T) {
	// given
	dir := t.TempDir()

	// when
	check := session.ExportCheckContinent(dir, false)

	// then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if check.Hint == "" {
		t.Error("expected hint for missing continent")
	}
	if !strings.Contains(check.Hint, "paintress init") {
		t.Errorf("hint should mention 'paintress init', got: %s", check.Hint)
	}
}

func TestCheckConfig_Missing_HasHint(t *testing.T) {
	// given
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0o755)

	// when
	check := session.ExportCheckConfig(dir)

	// then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if check.Hint == "" {
		t.Error("expected hint for missing config")
	}
	if !strings.Contains(check.Hint, "paintress init") {
		t.Errorf("hint should mention 'paintress init', got: %s", check.Hint)
	}
}

func TestCheckGitRepo_NotRepo_HasHint(t *testing.T) {
	// given
	dir := t.TempDir()

	// when
	check := session.ExportCheckGitRepo(dir)

	// then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if check.Hint == "" {
		t.Error("expected hint for non-git directory")
	}
	if !strings.Contains(check.Hint, "git init") {
		t.Errorf("hint should mention 'git init', got: %s", check.Hint)
	}
}

func TestCheckWritability_NotWritable_HasHint(t *testing.T) {
	// given — no .expedition/ means probe write fails
	dir := t.TempDir()

	// when
	check := session.ExportCheckWritability(dir)

	// then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if check.Hint == "" {
		t.Error("expected hint for not writable")
	}
}

func TestCheckSkills_NotFound_HasHint(t *testing.T) {
	// given
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".expedition"), 0o755)

	// when
	check := session.ExportCheckSkills(dir)

	// then
	if check.Status == domain.CheckOK {
		t.Fatal("expected fail")
	}
	if check.Hint == "" {
		t.Error("expected hint for missing skills")
	}
	if !strings.Contains(check.Hint, "paintress init") {
		t.Errorf("hint should mention 'paintress init', got: %s", check.Hint)
	}
}

func TestRunDoctor_BinaryNotFound_HasHint(t *testing.T) {
	// given: use a nonexistent claude command
	checks := session.RunDoctor("nonexistent-claude-xyz-99999", "", false)

	// then: the claude check should have a hint
	for _, c := range checks {
		if c.Name == "nonexistent-claude-xyz-99999" && c.Status == domain.CheckFail {
			if c.Hint == "" {
				t.Error("expected hint for missing required binary")
			}
			if !strings.Contains(c.Hint, "install") {
				t.Errorf("hint should mention install, got: %s", c.Hint)
			}
			return
		}
	}
	t.Error("expected to find failing check for nonexistent claude binary")
}
