// Package filter defines LLM action spaces: prompt templates,
// response schemas, and variable specifications.
package filter

import (
	"embed"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed prompts/*.yaml
var promptsFS embed.FS

// promptFile is the on-disk YAML schema for a prompt template.
type promptFile struct {
	Name        string            `yaml:"name"`
	Version     string            `yaml:"version"`
	Description string            `yaml:"description"`
	Variables   map[string]string `yaml:"variables"` // key → documentation
	Template    string            `yaml:"template"`
}

// PromptConfig is the read-only view of a loaded prompt template.
type PromptConfig struct { // nosemgrep: structure.multiple-exported-structs-go -- prompt registry family (PromptConfig/PromptRegistry) is a cohesive template loading set; splitting would break NewRegistryFromFS locality [permanent]
	Name        string
	Version     string
	Description string
	Variables   map[string]string // key → documentation
	Template    string
}

// PromptRegistry holds all embedded prompt templates keyed by name.
type PromptRegistry struct {
	entries map[string]PromptConfig
}

// NewRegistry loads all YAML files from the embedded prompts/ directory
// and returns a ready-to-use PromptRegistry.
func NewRegistry() (*PromptRegistry, error) {
	return NewRegistryFromFS(promptsFS)
}

// NewRegistryFromFS is the testable constructor that loads prompts from any fs.FS.
func NewRegistryFromFS(fsys fs.FS) (*PromptRegistry, error) {
	r := &PromptRegistry{entries: make(map[string]PromptConfig)}

	err := fs.WalkDir(fsys, "prompts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		var pf promptFile
		if unmarshalErr := yaml.Unmarshal(data, &pf); unmarshalErr != nil {
			return fmt.Errorf("parse %s: %w", path, unmarshalErr)
		}
		if pf.Name == "" {
			return fmt.Errorf("prompt file %s missing required 'name' field", path)
		}
		if pf.Template == "" {
			return fmt.Errorf("prompt file %s missing required 'template' field", path)
		}
		if _, dup := r.entries[pf.Name]; dup {
			return fmt.Errorf("duplicate prompt name %q in %s", pf.Name, path)
		}
		r.entries[pf.Name] = PromptConfig{
			Name:        pf.Name,
			Version:     pf.Version,
			Description: pf.Description,
			Variables:   pf.Variables,
			Template:    pf.Template,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("load prompt registry: %w", err)
	}
	if len(r.entries) == 0 {
		return nil, fmt.Errorf("load prompt registry: no prompt files found")
	}
	return r, nil
}

// Get returns the PromptConfig for the given name, or an error if not found.
func (r *PromptRegistry) Get(name string) (PromptConfig, error) {
	e, ok := r.entries[name]
	if !ok {
		return PromptConfig{}, fmt.Errorf("prompt %q not found in registry", name)
	}
	return e, nil
}

// Expand renders the named prompt template with the given variables.
// Variables use simple {key} placeholder syntax.
func (r *PromptRegistry) Expand(name string, vars map[string]string) (string, error) {
	e, err := r.Get(name)
	if err != nil {
		return "", err
	}
	return ExpandTemplate(e.Template, vars), nil
}

// MustExpand renders the named prompt template, panicking on error.
// Use in paths where the prompt name is known at compile time.
func (r *PromptRegistry) MustExpand(name string, vars map[string]string) string {
	s, err := r.Expand(name, vars)
	if err != nil {
		panic("filter.MustExpand: " + err.Error())
	}
	return s
}

// Names returns all registered prompt names (sorted for determinism).
func (r *PromptRegistry) Names() []string {
	names := make([]string, 0, len(r.entries))
	for n := range r.entries {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// ExpandTemplate performs simple {key} → value replacement.
func ExpandTemplate(tmpl string, vars map[string]string) string {
	// Phase 1: Process {#if key}...{#else}...{/if} conditionals.
	// A key is truthy if present in vars AND not empty/"false".
	result := processConditionals(tmpl, vars)

	// Phase 2: Two-pass expansion prevents variable values containing {key}
	// patterns from being re-expanded. Pass 1: placeholder → token.
	// Pass 2: token → value.
	const sentinel = "\x00PROMPT_VAR_"
	for k := range vars {
		result = strings.ReplaceAll(result, "{"+k+"}", sentinel+k+"\x00")
	}
	for k, v := range vars {
		result = strings.ReplaceAll(result, sentinel+k+"\x00", v)
	}
	return result
}

// processConditionals handles {#if key}...{#else}...{/if} blocks.
// Truthy: key exists in vars, value is non-empty and not "false".
// {#else} is optional. Blocks can be nested.
func processConditionals(tmpl string, vars map[string]string) string {
	for {
		start := strings.Index(tmpl, "{#if ")
		if start == -1 {
			return tmpl
		}
		closeTag := strings.Index(tmpl[start:], "}")
		if closeTag == -1 {
			return tmpl
		}
		key := tmpl[start+len("{#if ") : start+closeTag]

		endTag := "{/if}"
		endIdx := strings.Index(tmpl[start:], endTag)
		if endIdx == -1 {
			return tmpl
		}
		endIdx += start

		body := tmpl[start+closeTag+1 : endIdx]

		var ifBlock, elseBlock string
		if elseIdx := strings.Index(body, "{#else}"); elseIdx != -1 {
			ifBlock = body[:elseIdx]
			elseBlock = body[elseIdx+len("{#else}"):]
		} else {
			ifBlock = body
		}

		val, exists := vars[key]
		truthy := exists && val != "" && val != "false"

		var replacement string
		if truthy {
			replacement = ifBlock
		} else {
			replacement = elseBlock
		}

		tmpl = tmpl[:start] + replacement + tmpl[endIdx+len(endTag):]
	}
}

// --- singleton ---

var (
	defaultRegistry     *PromptRegistry
	defaultRegistryOnce sync.Once
	defaultRegistryErr  error
)

// Default returns the package-level PromptRegistry singleton.
// It is loaded once from embedded YAML files and safe for concurrent use.
func Default() (*PromptRegistry, error) {
	defaultRegistryOnce.Do(func() {
		defaultRegistry, defaultRegistryErr = NewRegistry()
	})
	return defaultRegistry, defaultRegistryErr
}

// MustDefault returns the package-level PromptRegistry singleton,
// panicking if the embedded YAML files cannot be loaded.
func MustDefault() *PromptRegistry {
	r, err := Default()
	if err != nil {
		panic("filter.MustDefault: " + err.Error())
	}
	return r
}
