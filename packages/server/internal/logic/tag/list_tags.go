package tag

import (
	"context"

	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
	"log/slog"
)

// ListTagsLogic contains the listTags use case.
type ListTagsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewListTagsLogic creates a ListTagsLogic.
func NewListTagsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTagsLogic {
	return &ListTagsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.tag.listtags"),
	}
}

// ListTags 查询标签列表
func (l *ListTagsLogic) ListTags(userID uuid.UUID, input *request.ListTagsRequest) (*response.PageListResponse[*response.TagResponse], error) {
	if l.svcCtx == nil || l.svcCtx.TagRepo == nil {
		return nil, xerr.BadRequest("标签仓储未初始化")
	}
	if input == nil {
		input = &request.ListTagsRequest{}
	}
	scope, err := tagScopeFromInput(input)
	if err != nil {
		return nil, err
	}
	page, pageSize := normalizeTagPage(input.Page, input.PageSize)
	rows, total, err := l.svcCtx.TagRepo.PageList(l.ctx, userID, repository.TagListQuery{
		Scope: string(scope),
		PageQuery: repository.PageQuery{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		return nil, err
	}
	docCounts, imageCounts, err := loadTagCounts(l.ctx, l.svcCtx.TagRepo, userID, tagIDs(rows))
	if err != nil {
		return nil, err
	}
	return mapper.TagsToPageListResponse(rows, total, page, pageSize, docCounts, imageCounts), nil
}

func normalizeTagPage(page int64, pageSize int64) (int64, int64) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	return page, pageSize
}
