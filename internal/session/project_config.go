package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
// Supported keys: tracker.team, tracker.project, lang.
func UpdateProjectConfig(continent string, key string, value string) error {
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
		cfg.Tracker.Team = value
	case "tracker.project":
		cfg.Tracker.Project = value
	case "tracker.cycle":
		cfg.Tracker.Cycle = value
	case "lang":
		if !domain.ValidLang(value) {
			return fmt.Errorf("invalid lang %q: must be ja or en", value)
		}
		cfg.Lang = value
	case "max_expeditions":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid max_expeditions %q: must be non-negative integer", value)
		}
		cfg.MaxExpeditions = n
	case "timeout_sec":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid timeout_sec %q: must be non-negative integer", value)
		}
		cfg.TimeoutSec = n
	case "model":
		cfg.Model = value
	case "base_branch":
		cfg.BaseBranch = value
	case "claude_cmd":
		cfg.ClaudeCmd = value
	case "dev_cmd":
		cfg.DevCmd = value
	case "dev_dir":
		cfg.DevDir = value
	case "dev_url":
		cfg.DevURL = value
	case "review_cmd":
		cfg.ReviewCmd = value
	case "workers":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid workers %q: must be non-negative integer", value)
		}
		cfg.Workers = n
	case "setup_cmd":
		cfg.SetupCmd = value
	case "no_dev":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid no_dev %q: must be true or false", value)
		}
		cfg.NoDev = b
	case "notify_cmd":
		cfg.NotifyCmd = value
	case "approve_cmd":
		cfg.ApproveCmd = value
	case "auto_approve":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid auto_approve %q: must be true or false", value)
		}
		cfg.AutoApprove = b
	case "max_retries":
		n, err := strconv.Atoi(value)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid max_retries %q: must be non-negative integer", value)
		}
		cfg.MaxRetries = n
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
	return os.WriteFile(path, data, 0644)
}
