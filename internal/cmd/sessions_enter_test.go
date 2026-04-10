package cmd

// white-box-reason: cobra command routing: tests sessions enter subcommand end-to-end

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

func setupSessionsEnterEnv(t *testing.T, providerSessionID, workDir string) (repoRoot string, recordID string) {
	t.Helper()
	repoRoot = t.TempDir()
	stateDir := filepath.Join(repoRoot, domain.StateDir)
	runDir := filepath.Join(stateDir, ".run")
	os.MkdirAll(runDir, 0755)

	// Project config with echo as fake provider
	cfgContent := "claude_cmd: echo\n"
	os.WriteFile(filepath.Join(stateDir, "config.yaml"), []byte(cfgContent), 0644)

	store, err := session.NewSQLiteCodingSessionStore(filepath.Join(runDir, "sessions.db"))
	if err != nil {
		t.Fatalf("create session store: %v", err)
	}
	defer store.Close()

	rec := domain.NewCodingSessionRecord(domain.ProviderClaudeCode, "test-model", workDir)
	rec.ProviderSessionID = providerSessionID
	if err := store.Save(context.Background(), rec); err != nil {
		t.Fatalf("save session record: %v", err)
	}
	return repoRoot, rec.ID
}

func TestSessionsEnter_ByRecordID(t *testing.T) {
	workDir := t.TempDir()
	repoRoot, recordID := setupSessionsEnterEnv(t, "provider-sess-001", workDir)

	var stdout bytes.Buffer
	rootCmd := NewRootCommand()
	rootCmd.SetArgs([]string{"sessions", "enter", "--path", repoRoot, recordID})
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("sessions enter failed: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "--resume provider-sess-001") {
		t.Errorf("expected --resume provider-sess-001, got: %q", output)
	}
}

func TestSessionsEnter_ByConfigFlag(t *testing.T) {
	workDir := t.TempDir()
	repoRoot := t.TempDir()
	stateDir := filepath.Join(repoRoot, domain.StateDir)
	runDir := filepath.Join(stateDir, ".run")
	os.MkdirAll(runDir, 0755)

	// Write config with custom claude_cmd
	customCfg := "claude_cmd: custom-claude-binary\n"
	configPath := filepath.Join(stateDir, "config.yaml")
	os.WriteFile(configPath, []byte(customCfg), 0644)

	store, err := session.NewSQLiteCodingSessionStore(filepath.Join(runDir, "sessions.db"))
	if err != nil {
		t.Fatalf("create session store: %v", err)
	}
	defer store.Close()

	rec := domain.NewCodingSessionRecord(domain.ProviderClaudeCode, "test-model", workDir)
	rec.ProviderSessionID = "provider-sess-config"
	if err := store.Save(context.Background(), rec); err != nil {
		t.Fatalf("save session record: %v", err)
	}

	var stderr bytes.Buffer
	rootCmd := NewRootCommand()
	// Use --config to point to the custom config explicitly
	rootCmd.SetArgs([]string{"sessions", "enter", "--config", configPath, rec.ID})
	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&stderr)

	// This will fail because custom-claude-binary doesn't exist, but
	// the error should reference the custom command, proving config was loaded.
	err = rootCmd.Execute()
	// We expect an error because the binary doesn't exist
	if err == nil {
		t.Fatal("expected error from non-existent custom-claude-binary")
	}
	errMsg := err.Error() + stderr.String()
	// The error should mention the custom binary, not the default "claude"
	if !strings.Contains(errMsg, "custom-claude-binary") {
		t.Errorf("expected error to reference custom-claude-binary from --config, got: %q", errMsg)
	}
}

func TestSessionsEnter_ByProviderID(t *testing.T) {
	workDir := t.TempDir()
	repoRoot, _ := setupSessionsEnterEnv(t, "provider-sess-002", workDir)

	var stdout bytes.Buffer
	rootCmd := NewRootCommand()
	rootCmd.SetArgs([]string{"sessions", "enter", "--path", repoRoot, "--provider-id", "provider-sess-002"})
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&bytes.Buffer{})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("sessions enter --provider-id failed: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "--resume provider-sess-002") {
		t.Errorf("expected --resume provider-sess-002, got: %q", output)
	}
}
