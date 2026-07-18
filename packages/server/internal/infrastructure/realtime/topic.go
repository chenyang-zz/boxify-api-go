package realtime

import "github.com/google/uuid"

const conversationTopicPrefix = "conversation:"

func ConversationTopic(conversationID uuid.UUID) string {
	return conversationTopicPrefix + conversationID.String()
}
