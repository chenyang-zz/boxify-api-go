package agentconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// GetAgentConfigLogic contains the getAgentConfig use case.
type GetAgentConfigLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewGetAgentConfigLogic creates a GetAgentConfigLogic.
func NewGetAgentConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAgentConfigLogic {
	return &GetAgentConfigLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.agentconfig.getagentconfig"),
	}
}

// GetAgentConfig 查询智能体配置
func (l *GetAgentConfigLogic) GetAgentConfig(userID uuid.UUID) (*response.AgentConfigResponse, error) {
	config, err := l.svcCtx.AgentConfigRepo.FindByUserID(l.ctx, userID)
	if err == nil {
		return mapper.AgentConfigToResponse(config), nil
	}

	if xerr.From(err).Kind != xerr.KindNotFound {
		return nil, err
	}

	config, err = l.svcCtx.AgentConfigRepo.Create(l.ctx, userID, &models.AgentConfig{})
	if err != nil {
		return nil, err
	}
	return mapper.AgentConfigToResponse(config), nil
}
