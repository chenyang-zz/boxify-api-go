package image

import (
	"context"
	"log/slog"
	"strings"

	ragsearch "github.com/boxify/api-go/internal/core/rag/search"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const defaultImageSearchTopK = 12

// SearchImagesLogic contains the searchImages use case.
type SearchImagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewSearchImagesLogic creates a SearchImagesLogic.
func NewSearchImagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchImagesLogic {
	return &SearchImagesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.image.searchimages"),
	}
}

// SearchImages 检索图片
func (l *SearchImagesLogic) SearchImages(userID uuid.UUID, input *request.SearchImageRequest) (*response.ListResponse[*response.SearchImageResponse], error) {
	if input == nil || strings.TrimSpace(input.Query) == "" {
		return nil, xerr.BadRequest("检索关键词不能为空")
	}
	if l.svcCtx == nil || l.svcCtx.RAGSearcher == nil {
		return nil, xerr.Internal("图片检索依赖未初始化", nil)
	}
	llmClient, err := svc.EmbeddingClient(l.ctx, l.svcCtx, userID)
	if err != nil {
		return nil, err
	}
	topK := int(input.TopK)
	if topK <= 0 {
		topK = defaultImageSearchTopK
	}
	searchResult, err := l.svcCtx.RAGSearcher.Search(l.ctx, input.Query,
		ragsearch.WithTopK(topK),
		ragsearch.WithFilters(imageSearchFilters(userID)),
		ragsearch.WithInputEmbedder(llmClient),
	)
	if err != nil {
		return nil, err
	}
	results := searchResult.Results
	out := make([]*response.SearchImageResponse, 0, len(results))
	for _, item := range results {
		out = append(out, &response.SearchImageResponse{
			ChunkID:    item.Source.ChunkID,
			Content:    item.Content,
			ImageName:  item.Source.Name,
			SourceID:   item.Source.SourceID,
			SourceType: item.Source.SourceType,
			KBID:       item.Source.KBID,
			Score:      item.Score,
		})
	}
	l.log.InfoContext(l.ctx, "图片检索完成",
		slog.String("user_id", userID.String()),
		slog.Int("result_count", len(out)),
	)
	return &response.ListResponse[*response.SearchImageResponse]{List: out}, nil
}

func imageSearchFilters(userID uuid.UUID) []any {
	return []any{
		map[string]any{"term": map[string]any{"user_id": userID.String()}},
		map[string]any{"term": map[string]any{"source_type": "image"}},
	}
}
