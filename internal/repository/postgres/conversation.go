/**
 * @Time   : 2026/6/27 15:50
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package postgres

import (
	"context"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/xerr"
	"gorm.io/gorm"
)

type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) repository.ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(ctx context.Context, conversation *models.Conversation) (*models.Conversation, error) {
	if err := r.db.WithContext(ctx).Create(conversation).Error; err != nil {
		return nil, xerr.Wrapf(err, "创建会话失败")
	}
	return conversation, nil
}

func (r *ConversationRepository) List(ctx context.Context, conversation *models.Conversation) ([]*models.Conversation, error) {
	var rows []*models.Conversation

	err := r.db.WithContext(ctx).Model(conversation).Where("user_id = ?", conversation.UserID).Order("updated_at DESC").Find(&rows).Error
	if err != nil {
		return nil, xerr.Wrapf(err, "查询会话列表失败")
	}

	return rows, nil
}
