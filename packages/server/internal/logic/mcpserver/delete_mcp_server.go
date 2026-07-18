package mcpserver

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
	"log/slog"
)

// DeleteMCPServerLogic contains the deleteMCPServer use case.
type DeleteMCPServerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteMCPServerLogic creates a DeleteMCPServerLogic.
func NewDeleteMCPServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMCPServerLogic {
	return &DeleteMCPServerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.mcpserver.deletemcpserver"),
	}
}

// DeleteMCPServer 删除mcp服务
func (l *DeleteMCPServerLogic) DeleteMCPServer(userID uuid.UUID, input *request.UriMCPServerIDRequest) error {
	mcpServerID, err := mcpServerIDFromInput(input)
	if err != nil {
		return err
	}

	return l.svcCtx.MCPServerRepo.Delete(l.ctx, userID, mcpServerID)
}
