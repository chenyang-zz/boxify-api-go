package tag

import (
	"context"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// DeleteTagLogic contains the deleteTag use case.
type DeleteTagLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteTagLogic creates a DeleteTagLogic.
func NewDeleteTagLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTagLogic {
	return &DeleteTagLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.tag.deletetag"),
	}
}

// DeleteTag 删除标签
func (l *DeleteTagLogic) DeleteTag(userID uuid.UUID, input *request.UriTagServerIDRequest) error {
	if l.svcCtx == nil || l.svcCtx.TagRepo == nil {
		return xerr.BadRequest("标签仓储未初始化")
	}
	if input == nil {
		return xerr.BadRequest("标签 ID 不能为空")
	}
	tagID, err := tagIDFromInput(input.ID)
	if err != nil {
		return err
	}
	affectedDocumentIDs, err := l.svcCtx.TagRepo.ListDocumentIDsByTag(l.ctx, userID, tagID)
	if err != nil {
		return err
	}
	if err := l.svcCtx.TagRepo.Delete(l.ctx, userID, tagID); err != nil {
		return err
	}
	syncDocumentChunkTagsBestEffort(l.ctx, l.svcCtx, l.log, userID, tagID, affectedDocumentIDs, "delete")
	l.log.InfoContext(l.ctx, "删除标签",
		slog.String("user_id", userID.String()),
		slog.String("tag_id", tagID.String()),
	)
	return nil
}
