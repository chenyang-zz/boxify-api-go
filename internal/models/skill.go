/**
 * @Time   : 2026/6/29 16:02
 * @Author : chenyangzhao542@gmail.com
 * @File   : skill.go
 **/

package models

import (
	"time"

	"github.com/google/uuid"
)

type Skill struct {
	ID          uuid.UUID      `gorm:"column:id;type:uuid;primaryKey"`
	UserID      uuid.UUID      `gorm:"column:user_id;type:uuid;not null;index"`
	User        User           `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Name        string         `gorm:"column:name;size:64;not null"`
	Description string         `gorm:"column:description;size:256;default:''"`
	Icon        string         `gorm:"column:icon;size:16;default:'🧩'"`
	Prompt      string         `gorm:"column:prompt;type:text;default:''"`       // 专属任务提示词，对话时与角色卡 system_prompt 叠加注入
	ToolKeys    StringList     `gorm:"column:tool_keys;type:jsonb;default:'[]'"` // 工具白名单：内置工具 key 列表。非空=只用这些；空列表=不限定（用全局工具配置）
	KBID        *uuid.UUID     `gorm:"column:kb_id;type:uuid;index"`
	KB          KnowledgeBase  `gorm:"foreignKey:KBID;references:ID;constraint:OnDelete:SET NULL"`
	Config      map[string]any `gorm:"column:config;type:jsonb;default:'{}'"`    // 轻量配置：{ quick_prompts: [str], few_shots: [{input, output}] }
	Enabled     bool           `gorm:"column:enabled;not null;default:true"`     // 是否在对话页技能选择器中显示（关闭则不占用对话框入口，避免技能多时拥挤）
	IsBuiltin   bool           `gorm:"column:is_builtin;not null;default:false"` // 是否由内置模板复制而来（标记用途，用户仍可改删）
	Sort        int            `gorm:"column:sort;not null;default:0"`           // 列表排序
	CreatedAt   time.Time      `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"column:updated_at;autoUpdateTime"`
}

func (Skill) TableName() string {
	return "skills"
}
