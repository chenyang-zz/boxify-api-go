package mcpserver

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// SyncMCPServerLogic contains the syncMCPServer use case.
type SyncMCPServerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewSyncMCPServerLogic creates a SyncMCPServerLogic.
func NewSyncMCPServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncMCPServerLogic {
	return &SyncMCPServerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.mcpserver.syncmcpserver"),
	}
}

// SyncMCPServer 同步mcp服务
func (l *SyncMCPServerLogic) SyncMCPServer(userID uuid.UUID, input *request.UriMCPServerIDRequest) (*response.MCPServerResponse, error) {
	_ = l
	return nil, nil
}
