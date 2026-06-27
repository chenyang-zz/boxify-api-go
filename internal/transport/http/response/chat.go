/**
 * @Time   : 2026/6/27 22:12
 * @Author : chenyangzhao542@gmail.com
 * @File   : chat.go
 **/

package response

import "github.com/google/uuid"

type BaseEvent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ErrorEvent struct {
	BaseEvent
	Message string `json:"content"`
}

type MetaEvent struct {
	BaseEvent
	ConversationID uuid.UUID `json:"conversation_id"`
	Title          string    `json:"title"`
}

func (e *BaseEvent) EventName() string {
	return e.Type
}
