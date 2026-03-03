package session

import "github.com/hironow/paintress/internal/domain"

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

// State returns the current materialized projection state.
func (p *ProjectionApplier) State() *ExpeditionState {
	return p.state
}
