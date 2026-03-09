package session

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"gopkg.in/yaml.v3"
)

// InitProject creates .expedition/config.yaml using the provided values
// directly (no interactive prompts). Also runs ValidateContinent to ensure
// directory structure is set up. If config.yaml already exists, existing
// values are preserved (merged over defaults), with CLI values winning.
func InitProject(repoPath, team, project string, w io.Writer) error {
	if w == nil {
		w = io.Discard
	}
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if err := ValidateContinent(absPath, nil); err != nil {
		return fmt.Errorf("continent validation: %w", err)
	}

	cfg := domain.DefaultProjectConfig()
	if team != "" {
		cfg.Tracker.Team = team
	}
	if project != "" {
		cfg.Tracker.Project = project
	}

	// Merge existing config if present and valid
	cfgPath := domain.ProjectConfigPath(absPath)
	existing, readErr := os.ReadFile(cfgPath)
	if readErr == nil && len(existing) > 0 {
		var existingMap map[string]any
		if yamlErr := yaml.Unmarshal(existing, &existingMap); yamlErr == nil {
			var defaultMap map[string]any
			defaultData, marshalErr := yaml.Marshal(cfg)
			if marshalErr != nil {
				return marshalErr
			}
			if err := yaml.Unmarshal(defaultData, &defaultMap); err != nil {
				return err
			}

			// existing values override defaults
			deepMerge(defaultMap, existingMap)

			// CLI flags override everything
			cliOverrides := make(map[string]any)
			if team != "" {
				cliOverrides["tracker"] = map[string]any{"team": team, "project": project}
			}
			deepMerge(defaultMap, cliOverrides)

			merged, err := yaml.Marshal(defaultMap)
			if err != nil {
				return err
			}
			var mergedCfg domain.ProjectConfig
			if err := yaml.Unmarshal(merged, &mergedCfg); err != nil {
				return err
			}
			cfg = mergedCfg
		}
		// Invalid YAML: fall through to write fresh defaults
	}

	if err := SaveProjectConfig(absPath, &cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Fprintf(w, "\nConfig saved to %s\n", domain.ProjectConfigPath(absPath))
	if cfg.Tracker.Team != "" {
		fmt.Fprintf(w, "  Linear team:    %s\n", cfg.Tracker.Team)
	}
	if cfg.Tracker.Project != "" {
		fmt.Fprintf(w, "  Linear project: %s\n", cfg.Tracker.Project)
	}
	return nil
}

// deepMerge merges src into dst recursively. src values override dst values.
func deepMerge(dst, src map[string]any) {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = sv
			continue
		}
		srcMap, srcOK := sv.(map[string]any)
		dstMap, dstOK := dv.(map[string]any)
		if srcOK && dstOK {
			deepMerge(dstMap, srcMap)
		} else {
			dst[k] = sv
		}
	}
}
