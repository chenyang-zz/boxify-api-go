package mcpserver

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// TroggleMCPServerLogic contains the troggleMCPServer use case.
type TroggleMCPServerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewTroggleMCPServerLogic creates a TroggleMCPServerLogic.
func NewTroggleMCPServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TroggleMCPServerLogic {
	return &TroggleMCPServerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.mcpserver.trogglemcpserver"),
	}
}

// ToggleMCPServer 切换mcp状态
func (l *TroggleMCPServerLogic) ToggleMCPServer(userID uuid.UUID, input *request.ToggleMCPServerRequest) (*response.MCPServerResponse, error) {
	mcpServerID, err := mcpServerIDFromInput(&input.UriMCPServerIDRequest)
	if err != nil {
		return nil, err
	}

	patch := &models.MCPServer{Enabled: *input.Enabled}
	mcpServer, err := l.svcCtx.MCPServerRepo.UpdateFields(
		l.ctx,
		userID,
		mcpServerID,
		patch,
		repository.NewMCPServerUpdateFields().Enabled(),
	)
	if err != nil {
		return nil, err
	}

	plainAuthConfig, err := decryptMCPAuthConfig(l.svcCtx.SecretCipher, mcpServer.AuthConfig)
	if err != nil {
		return nil, err
	}
	return mapper.MCPServerToResponse(mcpServer, maskMCPAuthConfig(mcpServer.AuthType, plainAuthConfig)), nil
}
