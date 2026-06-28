package memory

import (
	"context"
	"sync"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/realtime"
)

type Broker struct {
	mu     sync.Mutex
	topics map[string]map[*subscription]struct{}
}

func New() realtime.Broker {
	return &Broker{topics: map[string]map[*subscription]struct{}{}}
}

func (b *Broker) Publish(ctx context.Context, topic string, event domain.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	for sub := range b.topics[topic] {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sub.events <- event:
		default:
		}
	}
	return nil
}

func (b *Broker) Subscribe(ctx context.Context, topic string) (realtime.Subscription, error) {
	sub := &subscription{
		broker: b,
		topic:  topic,
		events: make(chan domain.Event, 16),
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if b.topics[topic] == nil {
		b.topics[topic] = map[*subscription]struct{}{}
	}
	b.topics[topic][sub] = struct{}{}
	return sub, nil
}

type subscription struct {
	broker *Broker
	topic  string
	events chan domain.Event
	once   sync.Once
}

func (s *subscription) Events() <-chan domain.Event {
	return s.events
}

func (s *subscription) Close(ctx context.Context) error {
	_ = ctx
	s.once.Do(func() {
		s.broker.mu.Lock()
		defer s.broker.mu.Unlock()
		delete(s.broker.topics[s.topic], s)
		if len(s.broker.topics[s.topic]) == 0 {
			delete(s.broker.topics, s.topic)
		}
		close(s.events)
	})
	return nil
}
