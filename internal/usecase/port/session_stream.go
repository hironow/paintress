package port

import (
	"context"

	"github.com/hironow/paintress/internal/domain"
)

// SessionStreamPublisher publishes session stream events to subscribers.
type SessionStreamPublisher interface { // nosemgrep: structure.multiple-exported-interfaces-go -- session stream port family (SessionStreamPublisher/SessionStreamSubscriber/SessionStreamBus) is a cohesive pub/sub API; splitting would fragment the bus contract [permanent]
	Publish(ctx context.Context, event domain.SessionStreamEvent)
}

// SessionStreamSubscriber receives session stream events.
type SessionStreamSubscriber interface { // nosemgrep: structure.multiple-exported-interfaces-go -- session stream port family; see SessionStreamPublisher [permanent]
	C() <-chan domain.SessionStreamEvent
	Close()
}

// SessionStreamBus manages pub/sub for session stream events.
type SessionStreamBus interface { // nosemgrep: structure.multiple-exported-interfaces-go -- session stream port family; see SessionStreamPublisher [permanent]
	SessionStreamPublisher
	Subscribe(bufSize int) SessionStreamSubscriber
	Close()
}
