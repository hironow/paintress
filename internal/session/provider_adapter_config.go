package session

import "github.com/hironow/paintress/internal/domain"

// ProviderAdapterConfig holds the class-wide configuration for creating a
// provider adapter. All AI coding tools accept this shape in NewTrackedRunner.
// Role-specific policies (retry, lazy singleton) are separate from this contract.
type ProviderAdapterConfig struct {
	Cmd        string // provider CLI command (e.g. "claude")
	Model      string // model name (e.g. "opus")
	TimeoutSec int    // per-invocation timeout (0 = context deadline only)
	BaseDir    string // repository root (state dir parent)
	ToolName   string // tool identifier for stream events
}

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
