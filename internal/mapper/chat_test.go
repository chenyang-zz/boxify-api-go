package mapper_test

import (
	"context"
	"testing"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

func TestEventToResponseMapsTextEvent(t *testing.T) {
	got := mapper.EventToResponse(types.NewTokenEvent("hello"))

	event, ok := got.(*response.BaseEvent)
	if !ok {
		t.Fatalf("EventToResponse type = %T, want *response.BaseEvent", got)
	}
	if event.Type != types.EventTypeToken || event.Text != "hello" || event.EventName() != types.EventTypeToken {
		t.Fatalf("event = %+v, want token hello", event)
	}
}

func TestEventToResponseMapsMetaEvent(t *testing.T) {
	conversationID := uuid.New()
	got := mapper.EventToResponse(types.NewMetaEvent(conversationID, "New Chat"))

	event, ok := got.(*response.MetaEvent)
	if !ok {
		t.Fatalf("EventToResponse type = %T, want *response.MetaEvent", got)
	}
	if event.Type != types.EventTypeMeta || event.ConversationID != conversationID || event.Title != "New Chat" || event.EventName() != types.EventTypeMeta {
		t.Fatalf("event = %+v, want meta payload", event)
	}
}

func TestEventToResponseMapsErrorEvent(t *testing.T) {
	got := mapper.EventToResponse(types.NewErrorEvent("boom"))

	event, ok := got.(*response.ErrorEvent)
	if !ok {
		t.Fatalf("EventToResponse type = %T, want *response.ErrorEvent", got)
	}
	if event.Type != types.EventTypeError || event.Message != "boom" || event.EventName() != types.EventTypeError {
		t.Fatalf("event = %+v, want error payload", event)
	}
}

func TestEventStreamToResponseMapsAndCloses(t *testing.T) {
	events := make(chan types.Event, 2)
	events <- types.NewTokenEvent("hello")
	events <- types.NewDoneEvent("ok")
	close(events)

	out := mapper.EventStreamToResponse(context.Background(), events)
	first, ok := <-out
	if !ok {
		t.Fatal("first response missing")
	}
	second, ok := <-out
	if !ok {
		t.Fatal("second response missing")
	}
	if _, ok := <-out; ok {
		t.Fatal("response channel remained open")
	}

	if first.EventName() != types.EventTypeToken || second.EventName() != types.EventTypeDone {
		t.Fatalf("events = %q/%q, want token/done", first.EventName(), second.EventName())
	}
}

func TestEventStreamToResponseMapsPingToComment(t *testing.T) {
	events := make(chan types.Event, 1)
	events <- types.NewPingEvent()
	close(events)

	out := mapper.EventStreamToResponse(context.Background(), events)
	got, ok := <-out
	if !ok {
		t.Fatal("response missing")
	}
	comment, ok := got.(*response.CommentEvent)
	if !ok {
		t.Fatalf("response type = %T, want *response.CommentEvent", got)
	}
	if comment.Comment() != "ping" {
		t.Fatalf("comment = %q, want ping", comment.Comment())
	}
}

func TestEventStreamToResponseMapsNilEventToError(t *testing.T) {
	events := make(chan types.Event, 1)
	events <- nil
	close(events)

	out := mapper.EventStreamToResponse(context.Background(), events)
	got, ok := <-out
	if !ok {
		t.Fatal("response missing")
	}
	event, ok := got.(*response.BaseEvent)
	if !ok {
		t.Fatalf("response type = %T, want *response.BaseEvent", got)
	}
	if event.EventName() != types.EventTypeError {
		t.Fatalf("event = %q, want error", event.EventName())
	}
}
