package session

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hironow/paintress/internal/domain"
	"gopkg.in/yaml.v3"
)

// InitProject creates .expedition/config.yaml using the provided values
// directly (no interactive prompts). Also runs ValidateContinent to ensure
// directory structure is set up. If config.yaml already exists, existing
// values are preserved (merged over defaults), with CLI values winning.
// Returns an InitResult recording what was created/updated.
func InitProject(repoPath, team, project string) (*InitResult, error) {
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	result, err := ValidateContinent(absPath, nil)
	if err != nil {
		return result, fmt.Errorf("continent validation: %w", err)
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
				return result, marshalErr
			}
			if err := yaml.Unmarshal(defaultData, &defaultMap); err != nil {
				return result, err
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
				return result, err
			}
			var mergedCfg domain.ProjectConfig
			if err := yaml.Unmarshal(merged, &mergedCfg); err != nil {
				return result, err
			}
			cfg = mergedCfg
		}
		// Invalid YAML: fall through to write fresh defaults
	}

	if err := SaveProjectConfig(absPath, &cfg); err != nil {
		return result, fmt.Errorf("save config: %w", err)
	}
	result.Add(domain.StateDir+"/config.yaml", InitUpdated, "")

	return result, nil
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
