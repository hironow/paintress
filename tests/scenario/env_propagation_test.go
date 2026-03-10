//go:build scenario

package scenario_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestScenario_L1_EnvPrefixedClaudeCmd verifies that env-prefixed claude_cmd
// values (e.g. "CLAUDE_CONFIG_DIR=/tmp/test-config claude") propagate env vars
// to the spawned claude process. Uses fake-claude's FAKE_CLAUDE_ENV_LOG_DIR
// mechanism to capture and assert the received environment.
func TestScenario_L1_EnvPrefixedClaudeCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("scenario tests are not short")
	}

	ws := NewWorkspace(t, "minimal")

	// fake-claude is built as "claude" in BinDir (see scenario_test.go buildAllBinaries).
	fakeClaude := filepath.Join(ws.BinDir, "claude")
	envLogDir := t.TempDir()

	// Rewrite .expedition/config.yaml with env-prefixed claude_cmd and no_dev=true.
	cfgPath := filepath.Join(ws.RepoPath, ".expedition", "config.yaml")
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg map[string]any
	if err := yaml.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}

	cfg["claude_cmd"] = "CLAUDE_CONFIG_DIR=/tmp/test-config " + fakeClaude
	cfg["no_dev"] = true

	updatedData, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(cfgPath, updatedData, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	ws.Env = append(ws.Env, "FAKE_CLAUDE_ENV_LOG_DIR="+envLogDir)

	// Run paintress doctor — it invokes claude --version, mcp list, and inference.
	err = ws.RunPaintress(t, context.Background(), "doctor", ws.RepoPath)
	if err != nil {
		t.Fatalf("paintress doctor failed: %v", err)
	}

	// Verify env propagation: fake-claude should have logged env vars.
	entries, err := os.ReadDir(envLogDir)
	if err != nil {
		t.Fatalf("read env log dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no env log files written — CLAUDE_CONFIG_DIR was not propagated")
	}

	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(envLogDir, entry.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		var logged map[string]any
		if err := json.Unmarshal(data, &logged); err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
		if v, ok := logged["CLAUDE_CONFIG_DIR"]; !ok || v != "/tmp/test-config" {
			t.Errorf("%s: CLAUDE_CONFIG_DIR = %v, want /tmp/test-config", entry.Name(), v)
		}
	}

	t.Logf("verified CLAUDE_CONFIG_DIR propagation in %d invocations", len(entries))
}
