package session

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/hironow/paintress"
	"gopkg.in/yaml.v3"
)

// LoadProjectConfig reads the project config from .expedition/config.yaml.
// Returns a zero-value config (no error) if the file does not exist.
func LoadProjectConfig(continent string) (*paintress.ProjectConfig, error) {
	data, err := os.ReadFile(paintress.ProjectConfigPath(continent))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &paintress.ProjectConfig{}, nil
		}
		return nil, err
	}

	var cfg paintress.ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveProjectConfig writes the project config to .expedition/config.yaml.
func SaveProjectConfig(continent string, cfg *paintress.ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	path := paintress.ProjectConfigPath(continent)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
