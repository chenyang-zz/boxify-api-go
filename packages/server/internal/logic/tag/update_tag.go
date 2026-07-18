package tag

import (
	"context"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// UpdateTagLogic contains the updateTag use case.
type UpdateTagLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewUpdateTagLogic creates a UpdateTagLogic.
func NewUpdateTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateTagLogic {
	return &UpdateTagLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.tag.updatetag"),
	}
}

// UpdateTag 更新标签
func (l *UpdateTagLogic) UpdateTag(userID uuid.UUID, input *request.TagUpdateRequest) (*response.TagResponse, error) {
	if l.svcCtx == nil || l.svcCtx.TagRepo == nil {
		return nil, xerr.BadRequest("标签仓储未初始化")
	}
	if input == nil {
		return nil, xerr.BadRequest("标签更新参数不能为空")
	}
	tagID, err := tagIDFromInput(input.ID)
	if err != nil {
		return nil, err
	}
	patch := &models.Tag{}
	fields := repository.NewTagUpdateFields()
	changed := make([]string, 0, 2)
	nameChanged := false
	if input.Name != nil {
		name, err := trimTagField(input.Name, "标签名称不能为空")
		if err != nil {
			return nil, err
		}
		patch.Name = name
		fields.Name()
		changed = append(changed, "name")
		nameChanged = true
	}
	if input.Color != nil {
		color, err := trimTagField(input.Color, "标签颜色不能为空")
		if err != nil {
			return nil, err
		}
		patch.Color = color
		fields.Color()
		changed = append(changed, "color")
	}
	if len(changed) == 0 {
		return nil, xerr.BadRequest("更新字段不能为空")
	}
	var affectedDocumentIDs []uuid.UUID
	if nameChanged {
		affectedDocumentIDs, err = l.svcCtx.TagRepo.ListDocumentIDsByTag(l.ctx, userID, tagID)
		if err != nil {
			return nil, err
		}
	}
	row, err := l.svcCtx.TagRepo.UpdateFields(l.ctx, userID, tagID, patch, fields)
	if err != nil {
		return nil, err
	}
	if nameChanged {
		syncDocumentChunkTagsBestEffort(l.ctx, l.svcCtx, l.log, userID, tagID, affectedDocumentIDs, "update")
	}
	docCounts, imageCounts, err := loadTagCounts(l.ctx, l.svcCtx.TagRepo, userID, []uuid.UUID{row.ID})
	if err != nil {
		return nil, err
	}
	l.log.InfoContext(l.ctx, "更新标签",
		slog.String("user_id", userID.String()),
		slog.String("tag_id", tagID.String()),
		slog.Any("fields", changed),
	)
	return mapper.TagToResponse(row, docCounts[row.ID], imageCounts[row.ID]), nil
}
