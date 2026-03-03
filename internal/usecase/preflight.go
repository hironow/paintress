package usecase

import (
	"github.com/hironow/paintress/internal/domain"
	"github.com/hironow/paintress/internal/session"
)

// PreflightCheck verifies that required binaries are available in PATH.
func PreflightCheck(binaries ...string) error {
	return session.PreflightCheck(binaries...)
}

// NewEventStore creates an event store for the given events directory.
func NewEventStore(eventsDir string) domain.EventStore {
	return session.NewEventStore(eventsDir)
}

// ValidateContinent ensures the .expedition directory structure exists.
func ValidateContinent(continent string) error {
	return session.ValidateContinent(continent)
}
