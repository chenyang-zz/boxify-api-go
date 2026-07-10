package toolconfig

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// ListToolConfigsLogic contains the listToolConfigs use case.
type ListToolConfigsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListToolConfigsLogic creates a ListToolConfigsLogic.
func NewListToolConfigsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListToolConfigsLogic {
	return &ListToolConfigsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.toolconfig.listtoolconfigs"),
	}
}

// ListToolConfigs 查询工具配置列表
func (l *ListToolConfigsLogic) ListToolConfigs(userID uuid.UUID) (*response.ListResponse[*response.ToolConfigResponse], error) {
	items, err := builtinToolResponses(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	rows, err := l.svcCtx.ToolConfigRepo.List(l.ctx, userID)
	if err != nil {
		return nil, err
	}

	// 仓储按更新时间倒序返回；重复配置只采用最新一条。
	enabledByKey := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		if _, exists := enabledByKey[row.ToolKey]; !exists {
			enabledByKey[row.ToolKey] = row.Enabled
		}
	}
	for _, item := range items {
		if enabled, ok := enabledByKey[item.ToolKey]; ok {
			item.Enabled = enabled
		}
	}
	return &response.ListResponse[*response.ToolConfigResponse]{List: items}, nil
}
