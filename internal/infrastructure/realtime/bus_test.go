package realtime_test

import (
	"context"
	"errors"
	"testing"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/infrastructure/realtime"
	realtimememory "github.com/boxify/api-go/internal/infrastructure/realtime/memory"
	"github.com/google/uuid"
)

func TestConversationTopic(t *testing.T) {
	conversationID := uuid.New()

	if got, want := realtime.ConversationTopic(conversationID), "conversation:"+conversationID.String(); got != want {
		t.Fatalf("ConversationTopic() = %q, want %q", got, want)
	}
}

func TestForwardRelaysEventsAndStopsOnDone(t *testing.T) {
	ctx := context.Background()
	broker := realtimememory.New()
	sub, err := broker.Subscribe(ctx, "topic")
	if err != nil {
		t.Fatalf("subscribe error = %v", err)
	}

	if err := broker.Publish(ctx, "topic", domain.NewTokenEvent("hello")); err != nil {
		t.Fatalf("publish token error = %v", err)
	}
	if err := broker.Publish(ctx, "topic", domain.NewDoneEvent("ok")); err != nil {
		t.Fatalf("publish done error = %v", err)
	}

	out := make(chan domain.Event, 2)
	if err := realtime.Forward(ctx, sub, out, realtime.ForwardOptions{}); err != nil {
		t.Fatalf("Forward error = %v", err)
	}

	first, ok := <-out
	if !ok {
		t.Fatal("first forwarded event missing")
	}
	second, ok := <-out
	if !ok {
		t.Fatal("second forwarded event missing")
	}
	if _, ok := <-out; ok {
		t.Fatal("out channel remained open")
	}
	if first.EventName() != domain.EventTypeToken || second.EventName() != domain.EventTypeDone {
		t.Fatalf("forwarded events = %q/%q, want token/done", first.EventName(), second.EventName())
	}
}

func TestForwardStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	broker := realtimememory.New()
	sub, err := broker.Subscribe(context.Background(), "topic")
	if err != nil {
		t.Fatalf("subscribe error = %v", err)
	}

	out := make(chan domain.Event)
	if err := realtime.Forward(ctx, sub, out, realtime.ForwardOptions{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Forward error = %v, want context.Canceled", err)
	}
	if _, ok := <-out; ok {
		t.Fatal("out channel remained open")
	}
}
