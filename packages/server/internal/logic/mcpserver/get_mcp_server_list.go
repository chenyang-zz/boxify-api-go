package mcpserver

import (
	"context"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// GetMCPServerListLogic contains the getMCPServerList use case.
type GetMCPServerListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewGetMCPServerListLogic creates a GetMCPServerListLogic.
func NewGetMCPServerListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMCPServerListLogic {
	return &GetMCPServerListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.mcpserver.getmcpserverlist"),
	}
}

// GetMCPServerList 查询mcp服务列表
func (l *GetMCPServerListLogic) GetMCPServerList(userID uuid.UUID) (*response.ListResponse[*response.MCPServerResponse], error) {
	rows, err := l.svcCtx.MCPServerRepo.List(l.ctx, userID)
	if err != nil {
		return nil, err
	}

	resList := make([]*response.MCPServerResponse, 0, len(rows))
	for _, row := range rows {
		authConfig, err := decryptMCPAuthConfig(l.svcCtx.SecretCipher, row.AuthConfig)
		if err != nil {
			l.log.WarnContext(l.ctx, "MCP认证配置解析失败", slog.String("错误", err.Error()))
			continue
		}
		resList = append(resList, mapper.MCPServerToResponse(row, maskMCPAuthConfig(row.AuthType, authConfig)))
	}

	return &response.ListResponse[*response.MCPServerResponse]{
		List: resList,
	}, nil
}
