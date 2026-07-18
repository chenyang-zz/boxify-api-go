/**
 * @Time   : 2026/6/29 16:30
 * @Author : chenyangzhao542@gmail.com
 * @File   : mcp.go
 **/

package response

import (
	"time"

	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

type MCPMeta struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type MCPServerResponse struct {
	ID         uuid.UUID             `json:"id"`
	Name       string                `json:"name"`
	Transport  request.TransportType `json:"transport"`
	Url        string                `json:"url"`
	AuthType   request.AuthType      `json:"auth_type"`
	AuthMasked string                `json:"auth_masked"`
	Enabled    bool                  `json:"enabled"`
	Status     string                `json:"status"`
	LastError  *string               `json:"last_error"`
	ToolsCache []*MCPMeta            `json:"tools_cache"`
	SyncedAt   *time.Time            `json:"synced_at"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}
