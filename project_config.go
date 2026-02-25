package paintress

import "path/filepath"

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
