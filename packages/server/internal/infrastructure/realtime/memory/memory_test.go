package memory

import (
	"context"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/realtime"
)

func TestBrokerBroadcastsToSubscribers(t *testing.T) {
	ctx := context.Background()
	broker := New()

	first, err := broker.Subscribe(ctx, "topic")
	if err != nil {
		t.Fatalf("first subscribe error = %v", err)
	}
	defer first.Close(ctx)
	second, err := broker.Subscribe(ctx, "topic")
	if err != nil {
		t.Fatalf("second subscribe error = %v", err)
	}
	defer second.Close(ctx)

	if err := broker.Publish(ctx, "topic", types.NewTokenEvent("hello")); err != nil {
		t.Fatalf("publish error = %v", err)
	}

	for name, sub := range map[string]realtime.Subscription{"first": first, "second": second} {
		select {
		case event := <-sub.Events():
			if event.EventName() != types.EventTypeToken {
				t.Fatalf("%s event = %q, want token", name, event.EventName())
			}
		case <-time.After(time.Second):
			t.Fatalf("%s subscriber did not receive event", name)
		}
	}
}

func TestSubscriptionCloseStopsReceivingEvents(t *testing.T) {
	ctx := context.Background()
	broker := New()
	sub, err := broker.Subscribe(ctx, "topic")
	if err != nil {
		t.Fatalf("subscribe error = %v", err)
	}
	if err := sub.Close(ctx); err != nil {
		t.Fatalf("close error = %v", err)
	}

	if err := broker.Publish(ctx, "topic", types.NewTokenEvent("hello")); err != nil {
		t.Fatalf("publish error = %v", err)
	}

	if _, ok := <-sub.Events(); ok {
		t.Fatal("subscription channel remained open after close")
	}
}
