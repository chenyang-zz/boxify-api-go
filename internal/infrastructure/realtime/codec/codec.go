package codec

import (
	"encoding/json"

	"github.com/boxify/api-go/internal/domain"
	"github.com/google/uuid"
)

type eventEnvelope struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type textEventData struct {
	Text string `json:"text"`
}

type metaEventData struct {
	ConversationID uuid.UUID `json:"conversation_id"`
	Title          string    `json:"title"`
}

type errorEventData struct {
	Message string `json:"message"`
}

func MarshalEvent(event domain.Event) ([]byte, error) {
	var data any = map[string]any{}
	switch e := event.(type) {
	case *domain.TextEvent:
		data = textEventData{Text: e.Text}
	case *domain.MetaEvent:
		data = metaEventData{ConversationID: e.ConversationID, Title: e.Title}
	case *domain.ErrorEvent:
		data = errorEventData{Message: e.Message}
	}

	return json.Marshal(struct {
		Event string `json:"event"`
		Data  any    `json:"data"`
	}{
		Event: event.EventName(),
		Data:  data,
	})
}

func UnmarshalEvent(payload []byte) (domain.Event, error) {
	var envelope eventEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}

	switch envelope.Event {
	case domain.EventTypeToken:
		var data textEventData
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return nil, err
		}
		return domain.NewTokenEvent(data.Text), nil
	case domain.EventTypeDone:
		var data textEventData
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return nil, err
		}
		return domain.NewDoneEvent(data.Text), nil
	case domain.EventTypeMeta:
		var data metaEventData
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return nil, err
		}
		return domain.NewMetaEvent(data.ConversationID, data.Title), nil
	case domain.EventTypeError:
		var data errorEventData
		if err := json.Unmarshal(envelope.Data, &data); err != nil {
			return nil, err
		}
		return domain.NewErrorEvent(data.Message), nil
	case domain.EventTypePing:
		return domain.NewPingEvent(), nil
	default:
		return domain.NewBaseEvent(envelope.Event), nil
	}
}
