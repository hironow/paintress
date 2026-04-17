package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hironow/paintress/internal/domain"
	"gopkg.in/yaml.v3"
)

// LoadProjectConfig reads the project config from .expedition/config.yaml.
// Returns a default config (with lang=ja) if the file does not exist.
func LoadProjectConfig(continent string) (*domain.ProjectConfig, error) {
	cfg := domain.DefaultProjectConfig()

	data, err := os.ReadFile(domain.ProjectConfigPath(continent))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateProjectConfig reads the project config, updates a single key, validates, and writes back.
// Supported keys: tracker.team, tracker.project, tracker.cycle, lang, max_expeditions,
// timeout_sec, model, base_branch, claude_cmd, dev_cmd, dev_dir, dev_url, review_cmd,
// workers, setup_cmd, no_dev, notify_cmd, approve_cmd, auto_approve, max_retries, idle_timeout.
func UpdateProjectConfig(continent string, key string, value string) error { // nosemgrep: domain-primitives.multiple-string-params-go -- continent/key/value are semantically distinct config params [permanent]
	cfg, err := LoadProjectConfig(continent)
	if err != nil {
		return err
	}

	if err := setProjectConfigField(cfg, key, value); err != nil {
		return err
	}

	// Validate before writing
	if errs := domain.ValidateProjectConfig(*cfg); len(errs) > 0 {
		return fmt.Errorf("invalid config after update: %s", errs[0])
	}

	return SaveProjectConfig(continent, cfg)
}

func setProjectConfigField(cfg *domain.ProjectConfig, key string, value string) error {
	switch key {
	case "tracker.team":
		cfg.Tracker.Team = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "tracker.project":
		cfg.Tracker.Project = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "tracker.cycle":
		cfg.Tracker.Cycle = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "lang":
		if !domain.ValidLang(value) {
			return fmt.Errorf("invalid lang %q: must be ja or en", value)
		}
		cfg.Lang = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "max_expeditions":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid max_expeditions %q: must be non-negative integer", value)
		}
		cfg.MaxExpeditions = n // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "timeout_sec":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid timeout_sec %q: must be non-negative integer", value)
		}
		cfg.TimeoutSec = n // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "model":
		cfg.Model = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "base_branch":
		cfg.BaseBranch = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "claude_cmd":
		cfg.ClaudeCmd = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "dev_cmd":
		cfg.DevCmd = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "dev_dir":
		cfg.DevDir = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "dev_url":
		cfg.DevURL = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "review_cmd":
		cfg.ReviewCmd = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "workers":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid workers %q: must be non-negative integer", value)
		}
		cfg.Workers = n // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "setup_cmd":
		cfg.SetupCmd = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "no_dev":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid no_dev %q: must be true or false", value)
		}
		cfg.NoDev = b // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "notify_cmd":
		cfg.NotifyCmd = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "approve_cmd":
		cfg.ApproveCmd = value // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "auto_approve":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid auto_approve %q: must be true or false", value)
		}
		cfg.AutoApprove = b // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "max_retries":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid max_retries %q: must be non-negative integer", value)
		}
		cfg.MaxRetries = n // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	case "idle_timeout":
		d, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("invalid idle_timeout %q: %w", value, err)
		}
		cfg.IdleTimeout = d // nosemgrep: immutability.no-pointer-field-mutation-go -- config setter pattern: mutation is intentional at config load time; immutable builder rewrite is over-engineering for CLI config [permanent]
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

// SaveProjectConfig writes the project config to .expedition/config.yaml.
func SaveProjectConfig(continent string, cfg *domain.ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	path := domain.ProjectConfigPath(continent)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	header := "# paintress configuration\n# Run 'paintress init --force' to regenerate with defaults\n\n"
	return os.WriteFile(path, []byte(header+string(data)), 0644)
}
