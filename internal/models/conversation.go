/**
 * @Time   : 2026/6/27 15:19
 * @Author : chenyangzhao542@gmail.com
 * @File   : conversation.go
 **/

package models

import (
	"time"

	"github.com/google/uuid"
)

type Conversation struct {
	ID               uuid.UUID  `gorm:"column:id;type:uuid;primaryKey"`
	UserID           uuid.UUID  `gorm:"column:user_id;type:uuid;not null;index"`
	User             User       `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Title            string     `gorm:"column:title;size:255;not null;default:'新对话'"`
	IsGroup          bool       `gorm:"column:is_group;not null;default:false"`
	MemberPersonaIDs StringList `gorm:"column:member_persona_ids;type:jsonb"`
	EnableTools      bool       `gorm:"column:enable_tools;not null;default:false"`
	JoinCode         *string    `gorm:"column:join_code;size:16;index"`
	CreatedAt        time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"column:updated_at;autoUpdateTime"`
}

func (Conversation) TableName() string {
	return "conversations"
}
