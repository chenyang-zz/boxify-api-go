package realtime

import (
	"context"

	"github.com/boxify/api-go/internal/domain/types"
)

type Broker interface {
	Publish(ctx context.Context, topic string, event types.Event) error
	Subscribe(ctx context.Context, topic string) (Subscription, error)
}

type Subscription interface {
	Events() <-chan types.Event
	Close(ctx context.Context) error
}
