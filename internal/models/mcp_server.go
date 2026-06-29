/**
 * @Time   : 2026/6/29 16:18
 * @Author : chenyangzhao542@gmail.com
 * @File   : mcp_server.go
 **/

package models

import (
	"time"

	"github.com/google/uuid"
)

// MCPServer ORM 模型 —— 用户配置的外部 MCP 服务。
//
// 每个 server 是一组远程工具的来源（远程 SSE / Streamable HTTP 传输）。
// 认证信息（token / api_key）用 Fernet 加密存 auth_config；接口返回掩码。
// 工具清单同步后缓存在 tools_cache，启停粒度为 server 级。

type MCPServer struct {
	ID         uuid.UUID        `gorm:"column:id;type:uuid;primaryKey"`
	UserID     uuid.UUID        `gorm:"column:user_id;type:uuid;not null;index"`
	User       User             `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	Name       string           `gorm:"column:name;size:128;not null"`
	Transport  string           `gorm:"column:transport;size:32;not null;default:'streamable_http'"`
	Url        string           `gorm:"column:url;size:512;not null"`
	AuthType   string           `gorm:"column:auth_type;size:16;not null;default:'none'"`
	AuthConfig map[string]any   `gorm:"column:auth_config;type:jsonb;"` // 认证敏感信息：{"token": "<Fernet密文>"} 或 {"header": "X-Api-Key", "key": "<密文>"}
	Enabled    bool             `gorm:"column:enabled;not null;default:true"`
	Status     string           `gorm:"column:status;size:16;not null;default:'unknown'"`
	LastError  string           `gorm:"column:last_error;size:1024;not null;default:''"`
	ToolsCache []map[string]any `gorm:"column:tools_cache;type:jsonb;"` // 同步下来的工具清单：[{"name","description"}]
	SyncedAt   *time.Time       `gorm:"column:synced_at"`
	CreatedAt  time.Time        `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time        `gorm:"column:updated_at;autoUpdateTime"`
}

func (MCPServer) TableName() string {
	return "mcp_servers"
}
