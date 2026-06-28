/**
 * @Time   : 2026/6/28 14:47
 * @Author : chenyangzhao542@gmail.com
 * @File   : agent_persona.go
 **/

package models

import (
	"time"

	"github.com/google/uuid"
)

type AgentPersona struct {
	ID           uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	UserID       uuid.UUID `gorm:"column:user_id;type:uuid;not null;index"`
	User         User      `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Name         string    `gorm:"column:name;size:64;not null"`
	AvatarKey    string    `gorm:"column:avatar_key;size:512"`
	SystemPrompt string    `gorm:"column:system_prompt;type:text"` // 人格提示词（人设/语气/口头禅），对话时作为 system message 注入
	Temperature  float64   `gorm:"column:temperature;not null;default:0.7"`
	IsActive     bool      `gorm:"column:is_active;not null;default:false"`     // 是否当前生效（每用户最多一条 true）
	InGroupOnly  bool      `gorm:"column:in_group_only;not null;default:false"` // 仅作为角色卡组成员存在（如内置场景拉入的角色），不在「单个角色」列表单独展示
	Sort         int       `gorm:"column:sort;not null;default:0"`              // 列表排序（预留）
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (AgentPersona) TableName() string {
	return "agent_personas"
}
