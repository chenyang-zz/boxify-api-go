package image

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// ListImagesLogic contains the listImages use case.
type ListImagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListImagesLogic creates a ListImagesLogic.
func NewListImagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListImagesLogic {
	return &ListImagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.image.listimages"),
	}
}

// ListImages 查询图片列表
func (l *ListImagesLogic) ListImages(userID uuid.UUID, input *request.ListImagesRequest) (*response.PageListResponse[*response.ImageResponse], error) {
	if input == nil {
		return nil, xerr.BadRequest("图片列表参数不能为空")
	}
	kbID, err := parseOptionalKBID(input.KBID)
	if err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.ImageRepo.PageList(l.ctx, userID, repository.ImageListQuery{
		KBID: kbID,
		Tag:  input.Tag,
		PageQuery: repository.PageQuery{
			Page:     input.Page,
			PageSize: input.PageSize,
		},
	})
	if err != nil {
		return nil, err
	}
	out := make([]*response.ImageResponse, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapper.ImageToResponse(row, nil, imageURL(l.svcCtx, row.FileKey)))
	}
	return &response.PageListResponse[*response.ImageResponse]{
		Total:    total,
		Page:     input.Page,
		PageSize: input.PageSize,
		List:     out,
	}, nil
}
