package realtime

import (
	"context"

	"github.com/boxify/api-go/internal/domain"
)

type Broker interface {
	Publish(ctx context.Context, topic string, event domain.Event) error
	Subscribe(ctx context.Context, topic string) (Subscription, error)
}

type Subscription interface {
	Events() <-chan domain.Event
	Close(ctx context.Context) error
}
