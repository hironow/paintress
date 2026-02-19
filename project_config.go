package main

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ProjectConfig holds project-scoped configuration stored in .expedition/config.yaml.
type ProjectConfig struct {
	Linear LinearConfig `yaml:"linear"`
}

// LinearConfig holds Linear integration settings.
type LinearConfig struct {
	Team    string `yaml:"team"`
	Project string `yaml:"project,omitempty"`
}

// ProjectConfigPath returns the path to the project config file.
func ProjectConfigPath(continent string) string {
	return filepath.Join(continent, ".expedition", "config.yaml")
}

// LoadProjectConfig reads the project config from .expedition/config.yaml.
// Returns a zero-value config (no error) if the file does not exist.
func LoadProjectConfig(continent string) (*ProjectConfig, error) {
	data, err := os.ReadFile(ProjectConfigPath(continent))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ProjectConfig{}, nil
		}
		return nil, err
	}

	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveProjectConfig writes the project config to .expedition/config.yaml.
func SaveProjectConfig(continent string, cfg *ProjectConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(ProjectConfigPath(continent), data, 0644)
}
