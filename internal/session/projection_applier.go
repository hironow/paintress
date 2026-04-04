package session

import (
	"encoding/json"

	"github.com/hironow/paintress/internal/domain"
)

// ProjectionApplier implements domain.EventApplier by delegating to the
// existing applyEvent logic used by ProjectState.
// This enables event replay for rebuild use cases.
type ProjectionApplier struct {
	state *ExpeditionState
}

// compile-time interface check
var _ domain.EventApplier = (*ProjectionApplier)(nil)

// NewProjectionApplier creates a ProjectionApplier with zeroed state.
func NewProjectionApplier() *ProjectionApplier {
	return &ProjectionApplier{state: &ExpeditionState{}}
}

// Apply applies a single event to the materialized projection state.
func (p *ProjectionApplier) Apply(event domain.Event) error {
	applyEvent(p.state, event)
	return nil
}

// Rebuild resets the projection state and replays all given events.
func (p *ProjectionApplier) Rebuild(events []domain.Event) error {
	p.state = &ExpeditionState{}
	for _, ev := range events {
		applyEvent(p.state, ev)
	}
	return nil
}

// Serialize returns the projection state as JSON bytes.
func (p *ProjectionApplier) Serialize() ([]byte, error) {
	return json.Marshal(p.state)
}

// Deserialize restores projection state from JSON bytes.
func (p *ProjectionApplier) Deserialize(data []byte) error {
	return json.Unmarshal(data, p.state)
}

// State returns the current materialized projection state.
func (p *ProjectionApplier) State() *ExpeditionState {
	return p.state
}
