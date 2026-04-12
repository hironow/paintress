// white-box-reason: tests unexported assembly helper for ProviderAdapterConfig field omission guard
package session

import (
	"reflect"
	"testing"

	"github.com/hironow/paintress/internal/domain"
)

func TestAdapterConfigFromProjectConfig_AllFieldsPopulated(t *testing.T) {
	cfg := domain.Config{
		ClaudeCmd:  "test-claude",
		TimeoutSec: 42,
		Continent:  "/test/base",
	}
	pac := AdapterConfigFromProjectConfig(cfg, "test-model")

	v := reflect.ValueOf(pac)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			t.Errorf("field %s is zero — helper omitted it", v.Type().Field(i).Name)
		}
	}
}
