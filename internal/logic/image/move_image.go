package image

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

// MoveImageLogic contains the moveImage use case.
type MoveImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewMoveImageLogic creates a MoveImageLogic.
func NewMoveImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MoveImageLogic {
	return &MoveImageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.image.moveimage"),
	}
}

// MoveImage 移动图片到指定知识库
func (l *MoveImageLogic) MoveImage(userID uuid.UUID, input *request.MoveImageRequest) (*response.ImageResponse, error) {
	imageID, err := parseImageID(input.ImageID)
	if err != nil {
		return nil, err
	}
	kbID, err := parseRequiredKBID(input.KBID)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.KnowledgeBaseRepo.FindByID(l.ctx, userID, kbID); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.ImageRepo.UpdateFields(l.ctx, userID, imageID, &models.Image{
		KBID: &kbID,
	}, repository.NewImageUpdateFields().KBID())
	if err != nil {
		return nil, err
	}
	l.log.InfoContext(l.ctx, "移动图片到知识库",
		slog.String("user_id", userID.String()),
		slog.String("image_id", imageID.String()),
		slog.String("kb_id", kbID.String()),
	)
	updateImageChunksKnowledgeBaseBestEffort(l.ctx, l.svcCtx, l.log, userID, imageID, kbID)
	return mapper.ImageToResponse(row, nil, imageURL(l.svcCtx, row.FileKey)), nil
}
