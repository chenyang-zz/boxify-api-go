/**
 * @Time   : 2026/6/27 22:56
 * @Author : chenyangzhao542@gmail.com
 * @File   : helper.go
 **/

package chat

import (
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func parseConversationID(id string) (uuid.UUID, error) {
	if id == "" {
		return uuid.Nil, xerr.BadRequest("会话 ID 无效")
	}
	conversationID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("会话 ID 无效")
	}
	return conversationID, nil
}
