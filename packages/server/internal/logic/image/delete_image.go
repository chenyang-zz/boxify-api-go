package image

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

// DeleteImageLogic contains the deleteImage use case.
type DeleteImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewDeleteImageLogic creates a DeleteImageLogic.
func NewDeleteImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteImageLogic {
	return &DeleteImageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.image.deleteimage"),
	}
}

// DeleteImage 删除图片
func (l *DeleteImageLogic) DeleteImage(userID uuid.UUID, input *request.UriImageIDRequest) error {
	imageID, err := parseImageID(input.ImageID)
	if err != nil {
		return err
	}
	row, err := l.svcCtx.ImageRepo.FindByID(l.ctx, userID, imageID)
	if err != nil {
		return err
	}
	if l.svcCtx.Storage != nil && row.FileKey != "" {
		if err := l.svcCtx.Storage.Delete(l.ctx, row.FileKey); err != nil {
			l.log.WarnContext(l.ctx, "删除图片存储文件失败（忽略）",
				slog.String("user_id", userID.String()),
				slog.String("image_id", imageID.String()),
				slog.String("file_key", row.FileKey),
				slog.String("error", err.Error()),
			)
		}
	}
	deleteImageChunksBestEffort(l.ctx, l.svcCtx, l.log, userID, imageID)
	if err := l.svcCtx.ImageRepo.Delete(l.ctx, userID, imageID); err != nil {
		return err
	}
	l.log.InfoContext(l.ctx, "删除图片",
		slog.String("user_id", userID.String()),
		slog.String("image_id", imageID.String()),
	)
	return nil
}
