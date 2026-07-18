package image

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
)

// GetImageLogic contains the getImage use case.
type GetImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewGetImageLogic creates a GetImageLogic.
func NewGetImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetImageLogic {
	return &GetImageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.image.getimage"),
	}
}

// GetImage 获取图片详情
func (l *GetImageLogic) GetImage(userID uuid.UUID, input *request.UriImageIDRequest) (*response.ImageResponse, error) {
	imageID, err := parseImageID(input.ImageID)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.ImageRepo.FindByID(l.ctx, userID, imageID)
	if err != nil {
		return nil, err
	}
	return mapper.ImageToResponse(row, nil, imageURL(l.svcCtx, row.FileKey)), nil
}
