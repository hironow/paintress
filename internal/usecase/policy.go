package usecase

import (
	"context"
	"fmt"

	"github.com/hironow/paintress"
)

// PolicyHandler processes a domain event as part of a policy reaction.
// WHEN [EVENT] THEN [handler logic].
type PolicyHandler func(ctx context.Context, event paintress.Event) error

// PolicyEngine dispatches domain events to registered policy handlers.
// This connects the POLICY registry (paintress.Policies) to executable handlers.
type PolicyEngine struct {
	handlers map[paintress.EventType][]PolicyHandler
	logger   *paintress.Logger
}

// NewPolicyEngine creates a PolicyEngine. Pass nil logger for silent operation.
func NewPolicyEngine(logger *paintress.Logger) *PolicyEngine {
	return &PolicyEngine{
		handlers: make(map[paintress.EventType][]PolicyHandler),
		logger:   logger,
	}
}

// Register adds a handler for the given event type.
// Multiple handlers can be registered for the same event type.
func (e *PolicyEngine) Register(trigger paintress.EventType, handler PolicyHandler) {
	e.handlers[trigger] = append(e.handlers[trigger], handler)
}

// Dispatch sends an event to all handlers registered for its type.
// Handlers execute sequentially; the first error stops dispatch.
func (e *PolicyEngine) Dispatch(ctx context.Context, event paintress.Event) error {
	handlers, ok := e.handlers[event.Type]
	if !ok {
		return nil
	}
	for _, h := range handlers {
		if err := h(ctx, event); err != nil {
			return fmt.Errorf("policy dispatch %s: %w", event.Type, err)
		}
	}
	return nil
}
