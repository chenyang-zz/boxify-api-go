package document

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

// SearchDocumentsLogic contains the searchDocuments use case.
type SearchDocumentsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewSearchDocumentsLogic creates a SearchDocumentsLogic.
func NewSearchDocumentsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchDocumentsLogic {
	return &SearchDocumentsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.document.searchdocuments"),
	}
}

// SearchDocuments 检索文档
func (l *SearchDocumentsLogic) SearchDocuments(userID uuid.UUID, input *request.SearchDocumentsRequest) (*response.ListResponse[*response.SearchDocumentResponse], error) {
	if input == nil || strings.TrimSpace(input.Query) == "" {
		return nil, xerr.BadRequest("检索关键词不能为空")
	}
	if l.svcCtx == nil || l.svcCtx.RAGSearcher == nil {
		return nil, xerr.Internal("文档检索依赖未初始化", nil)
	}
	llmClient, err := svc.EmbeddingClient(l.ctx, l.svcCtx, userID)
	if err != nil {
		return nil, err
	}
	searchResult, err := l.svcCtx.RAGSearcher.Search(l.ctx, input.Query,
		ragsearch.WithTopK(int(input.TopK)),
		ragsearch.WithFilters(documentSearchFilters(userID, input.Tags)),
		ragsearch.WithInputEmbedder(llmClient),
	)
	if err != nil {
		return nil, err
	}
	results := searchResult.Results
	out := make([]*response.SearchDocumentResponse, 0, len(results))
	for _, item := range results {
		out = append(out, &response.SearchDocumentResponse{
			ChunkID:    item.Source.ChunkID,
			Content:    item.Content,
			DocName:    item.Source.Name,
			SourceID:   item.Source.SourceID,
			SourceType: item.Source.SourceType,
			KBID:       item.Source.KBID,
			Score:      item.Score,
		})
	}
	l.log.InfoContext(l.ctx, "文档检索完成",
		slog.String("user_id", userID.String()),
		slog.Int("result_count", len(out)),
	)
	return &response.ListResponse[*response.SearchDocumentResponse]{List: out}, nil
}

func documentSearchFilters(userID uuid.UUID, tags []string) []any {
	filters := []any{map[string]any{"term": map[string]any{"user_id": userID.String()}}}
	cleanTags := make([]string, 0, len(tags))
	for _, tag := range tags {
		if value := strings.TrimSpace(tag); value != "" {
			cleanTags = append(cleanTags, value)
		}
	}
	if len(cleanTags) != 0 {
		filters = append(filters, map[string]any{"terms": map[string]any{"tags": cleanTags}})
	}
	return filters
}
