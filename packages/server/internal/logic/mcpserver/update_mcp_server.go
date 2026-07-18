package mcpserver

import (
	"context"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/google/uuid"
	"log/slog"
)

// UpdateMCPServerLogic contains the updateMCPServer use case.
type UpdateMCPServerLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewUpdateMCPServerLogic creates a UpdateMCPServerLogic.
func NewUpdateMCPServerLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMCPServerLogic {
	return &UpdateMCPServerLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.mcpserver.updatemcpserver"),
	}
}

// UpdateMCPServer 更新mcp服务
func (l *UpdateMCPServerLogic) UpdateMCPServer(userID uuid.UUID, input *request.UpdateMCPServerRequest) (*response.MCPServerResponse, error) {
	var uriInput *request.UriMCPServerIDRequest
	if input != nil {
		uriInput = &input.UriMCPServerIDRequest
	}
	mcpServerID, err := mcpServerIDFromInput(uriInput)
	if err != nil {
		return nil, err
	}

	patch := &models.MCPServer{}
	fields := repository.NewMCPServerUpdateFields()
	if input.Name != nil {
		util.AssignIfNotNil(&patch.Name, input.Name)
		fields.Name()
	}
	if input.Transport != nil {
		patch.Transport = string(*input.Transport)
		fields.Transport()
	}
	if input.Url != nil {
		util.AssignIfNotNil(&patch.Url, input.Url)
		fields.Url()
	}
	if input.AuthType != nil {
		patch.AuthType = string(*input.AuthType)
		fields.AuthType()
	}
	if input.AuthConfig != nil {
		authConfig, err := encryptMCPAuthConfig(l.svcCtx.SecretCipher, input.AuthConfig)
		if err != nil {
			return nil, err
		}
		patch.AuthConfig = authConfig
		fields.AuthConfig()
	}
	if input.Enabled != nil {
		util.AssignIfNotNil(&patch.Enabled, input.Enabled)
		fields.Enabled()
	}

	mcpServer, err := l.svcCtx.MCPServerRepo.UpdateFields(l.ctx, userID, mcpServerID, patch, fields)
	if err != nil {
		return nil, err
	}

	plainAuthConfig, err := decryptMCPAuthConfig(l.svcCtx.SecretCipher, mcpServer.AuthConfig)
	if err != nil {
		return nil, err
	}
	return mapper.MCPServerToResponse(mcpServer, maskMCPAuthConfig(mcpServer.AuthType, plainAuthConfig)), nil
}
