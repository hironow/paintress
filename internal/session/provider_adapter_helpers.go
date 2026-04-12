package session

import "github.com/hironow/paintress/internal/domain"

// AdapterConfigFromProjectConfig extracts ProviderAdapterConfig from a domain.Config.
// Model is passed separately because paintress overrides it per-expedition.
func AdapterConfigFromProjectConfig(cfg domain.Config, model string) ProviderAdapterConfig {
	return ProviderAdapterConfig{
		Cmd:        cfg.ClaudeCmd,
		Model:      model,
		TimeoutSec: cfg.TimeoutSec,
		BaseDir:    cfg.Continent,
		ToolName:   "paintress",
	}
}
