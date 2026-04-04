package filter

import "sync"

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
