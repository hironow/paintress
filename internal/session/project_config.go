package session

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

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

	switch key {
	case "tracker.team":
		cfg.Tracker.Team = value
	case "tracker.project":
		cfg.Tracker.Project = value
	case "lang":
		if !domain.ValidLang(value) {
			return fmt.Errorf("invalid lang %q: must be ja or en", value)
		}
		cfg.Lang = value
	default:
		return fmt.Errorf("unknown config key %q", key)
	}

	// Validate before writing
	if errs := domain.ValidateProjectConfig(*cfg); len(errs) > 0 {
		return fmt.Errorf("invalid config after update: %s", errs[0])
	}

	return SaveProjectConfig(continent, cfg)
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
