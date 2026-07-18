package mcpserver

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// CreateMCPServerLogic contains the createMCPServer use case.
type CreateMCPServerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewCreateMCPServerLogic creates a CreateMCPServerLogic.
func NewCreateMCPServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMCPServerLogic {
	return &CreateMCPServerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.mcpserver.createmcpserver"),
	}
}

// CreateMCPServer 新建mcp服务
func (l *CreateMCPServerLogic) CreateMCPServer(userID uuid.UUID, input *request.CreateMCPServerRequest) (*response.MCPServerResponse, error) {
	_, err := l.svcCtx.MCPServerRepo.FindByName(l.ctx, userID, input.Name)
	if err == nil {
		return nil, xerr.BadRequest("同名MCP服务已存在")
	}
	if xerr.From(err).Kind != xerr.KindNotFound {
		return nil, err
	}

	authConfig, err := encryptMCPAuthConfig(l.svcCtx.SecretCipher, input.AuthConfig)
	if err != nil {
		return nil, err
	}
	mcpServer := &models.MCPServer{
		Name:       input.Name,
		Transport:  "streamable_http",
		Url:        input.Url,
		AuthType:   "none",
		AuthConfig: authConfig,
		Enabled:    true,
	}
	if input.Transport != "" {
		mcpServer.Transport = string(input.Transport)
	}
	if input.AuthType != "" {
		mcpServer.AuthType = string(input.AuthType)
	}
	if input.Enabled != nil {
		mcpServer.Enabled = *input.Enabled
	}

	mcpServer, err = l.svcCtx.MCPServerRepo.Create(l.ctx, userID, mcpServer)
	if err != nil {
		return nil, err
	}

	plainAuthConfig := models.MCPAuthConfig{}
	if input.AuthConfig != nil {
		plainAuthConfig["token"] = input.AuthConfig.Token
		plainAuthConfig["header"] = input.AuthConfig.Header
		plainAuthConfig["key"] = input.AuthConfig.Key
	}
	return mapper.MCPServerToResponse(mcpServer, maskMCPAuthConfig(mcpServer.AuthType, plainAuthConfig)), nil
}
