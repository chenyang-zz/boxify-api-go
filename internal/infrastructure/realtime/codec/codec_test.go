package codec

import (
	"testing"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/google/uuid"
)

func TestEventCodecRoundTripsKnownEvents(t *testing.T) {
	conversationID := uuid.New()
	tests := []types.Event{
		types.NewTokenEvent("tok"),
		types.NewDoneEvent("done"),
		types.NewMetaEvent(conversationID, "Title"),
		types.NewErrorEvent("boom"),
		types.NewPingEvent(),
	}

	for _, event := range tests {
		payload, err := MarshalEvent(event)
		if err != nil {
			t.Fatalf("MarshalEvent(%q) error = %v", event.EventName(), err)
		}
		got, err := UnmarshalEvent(payload)
		if err != nil {
			t.Fatalf("UnmarshalEvent(%q) error = %v", event.EventName(), err)
		}
		if got.EventName() != event.EventName() {
			t.Fatalf("round trip event = %q, want %q", got.EventName(), event.EventName())
		}
	}
}

func TestEventCodecMapsUnknownEventsToBaseEvent(t *testing.T) {
	got, err := UnmarshalEvent([]byte(`{"event":"custom","data":{"x":1}}`))
	if err != nil {
		t.Fatalf("UnmarshalEvent error = %v", err)
	}
	if got.EventName() != "custom" {
		t.Fatalf("EventName() = %q, want custom", got.EventName())
	}
	if _, ok := got.(*types.BaseEvent); !ok {
		t.Fatalf("event type = %T, want *types.BaseEvent", got)
	}
}

func TestEventCodecRejectsInvalidJSON(t *testing.T) {
	if _, err := UnmarshalEvent([]byte(`{"event":`)); err == nil {
		t.Fatal("UnmarshalEvent returned nil error for invalid JSON")
	}
}
