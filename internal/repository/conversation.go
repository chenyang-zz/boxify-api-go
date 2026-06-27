/**
 * @Time   : 2026/6/27 15:48
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package repository

import (
	"context"

	"github.com/boxify/api-go/internal/models"
)

type ConversationRepository interface {
	Create(ctx context.Context, conversation *models.Conversation) (*models.Conversation, error)
	List(ctx context.Context, conversation *models.Conversation) ([]*models.Conversation, error)
}
