package redis

import (
	"context"
	"log/slog"
	"sync"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/realtime"
	"github.com/boxify/api-go/internal/infrastructure/realtime/codec"
	goredis "github.com/redis/go-redis/v9"
)

type Broker struct {
	client *goredis.Client
	log    *slog.Logger
}

func New(client *goredis.Client) realtime.Broker {
	return &Broker{
		client: client,
		log:    slog.Default(),
	}
}

func (b *Broker) Publish(ctx context.Context, topic string, event domain.Event) error {
	payload, err := codec.MarshalEvent(event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, topic, payload).Err()
}

func (b *Broker) Subscribe(ctx context.Context, topic string) (realtime.Subscription, error) {
	pubsub := b.client.Subscribe(ctx, topic)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}

	sub := &subscription{
		pubsub: pubsub,
		events: make(chan domain.Event, 16),
		log:    b.log,
	}
	go sub.run(ctx)
	return sub, nil
}

type subscription struct {
	pubsub *goredis.PubSub
	events chan domain.Event
	log    *slog.Logger
	once   sync.Once
}

func (s *subscription) Events() <-chan domain.Event {
	return s.events
}

func (s *subscription) Close(ctx context.Context) error {
	_ = ctx
	var err error
	s.once.Do(func() {
		err = s.pubsub.Close()
	})
	return err
}

func (s *subscription) run(ctx context.Context) {
	defer close(s.events)
	ch := s.pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-ch:
			if !ok {
				return
			}
			event, err := codec.UnmarshalEvent([]byte(message.Payload))
			if err != nil {
				s.log.WarnContext(ctx, "解析实时事件失败", slog.String("error", err.Error()))
				continue
			}
			select {
			case <-ctx.Done():
				return
			case s.events <- event:
			}
		}
	}
}
