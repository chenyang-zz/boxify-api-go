package tag

import (
	"context"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// MergeTagLogic contains the mergeTag use case.
type MergeTagLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewMergeTagLogic creates a MergeTagLogic.
func NewMergeTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MergeTagLogic {
	return &MergeTagLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.tag.mergetag"),
	}
}

// MergeTag 合并标签
func (l *MergeTagLogic) MergeTag(userID uuid.UUID, input *request.TagMergeRequest) (*response.TagResponse, error) {
	if l.svcCtx == nil || l.svcCtx.TagRepo == nil {
		return nil, xerr.BadRequest("标签仓储未初始化")
	}
	if input == nil {
		return nil, xerr.BadRequest("标签合并参数不能为空")
	}
	sourceID, err := tagIDFromInput(input.SourceID)
	if err != nil {
		return nil, err
	}
	targetID, err := tagIDFromInput(input.TargetID)
	if err != nil {
		return nil, err
	}
	if sourceID == targetID {
		return nil, xerr.BadRequest("不能合并相同标签")
	}
	affectedDocumentIDs, err := l.svcCtx.TagRepo.ListDocumentIDsByTag(l.ctx, userID, sourceID)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.TagRepo.Merge(l.ctx, userID, sourceID, targetID)
	if err != nil {
		return nil, err
	}
	syncDocumentChunkTagsBestEffort(l.ctx, l.svcCtx, l.log, userID, sourceID, affectedDocumentIDs, "merge")
	docCounts, imageCounts, err := loadTagCounts(l.ctx, l.svcCtx.TagRepo, userID, []uuid.UUID{row.ID})
	if err != nil {
		return nil, err
	}
	l.log.InfoContext(l.ctx, "合并标签",
		slog.String("user_id", userID.String()),
		slog.String("source_tag_id", sourceID.String()),
		slog.String("target_tag_id", targetID.String()),
	)
	return mapper.TagToResponse(row, docCounts[row.ID], imageCounts[row.ID]), nil
}
