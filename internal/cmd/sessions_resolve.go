package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const sessionsToolName = "paintress"

// resolveSessionsDir resolves the repository root and state directory for sessions commands.
// Resolution order: --path flag -> --config flag -> cwd.
func resolveSessionsDir(cmd *cobra.Command) (repoRoot, stateDirPath string, err error) {
	if cmd.Flags().Changed("path") {
		p, _ := cmd.Flags().GetString("path")
		repoRoot, err = filepath.Abs(p)
		if err != nil {
			return "", "", fmt.Errorf("resolve path: %w", err)
		}
	} else if f := cmd.Flags().Lookup("config"); f != nil && cmd.Flags().Changed("config") {
		configAbs, absErr := filepath.Abs(f.Value.String())
		if absErr != nil {
			return "", "", fmt.Errorf("resolve config path: %w", absErr)
		}
		repoRoot = filepath.Dir(filepath.Dir(configAbs))
	} else {
		repoRoot, err = os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("resolve cwd: %w", err)
		}
	}
	stateDirPath = filepath.Join(repoRoot, domain.StateDir)
	if info, statErr := os.Stat(stateDirPath); statErr != nil || !info.IsDir() {
		return "", "", fmt.Errorf("state directory not found: %s (run '%s init' first)", stateDirPath, sessionsToolName)
	}
	return repoRoot, stateDirPath, nil
}

// sessionsConfig holds the minimal config fields needed by sessions enter.
type sessionsConfig struct {
	ClaudeCmd string `yaml:"claude_cmd"`
}

// loadSessionsConfig reads config for sessions enter.
// Missing file → default ClaudeCmd (graceful). Malformed YAML → error (fail-fast).
func loadSessionsConfig(path string) (*sessionsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &sessionsConfig{ClaudeCmd: domain.DefaultClaudeCmd}, nil
		}
		return nil, err
	}
	cfg := &sessionsConfig{ClaudeCmd: domain.DefaultClaudeCmd}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.ClaudeCmd == "" {
		cfg.ClaudeCmd = domain.DefaultClaudeCmd
	}
	return cfg, nil
}
