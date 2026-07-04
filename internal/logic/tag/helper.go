package tag

import (
	"context"
	"log/slog"
	"strings"

	"github.com/boxify/api-go/internal/domain"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func tagIDFromInput(raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		return uuid.Nil, xerr.BadRequest("标签 ID 无效")
	}
	return id, nil
}

func tagScopeFromInput(input *request.ListTagsRequest) (domain.TagScope, error) {
	if input == nil || input.Scope == nil {
		return domain.TagScopeAll, nil
	}
	scope := strings.TrimSpace(*input.Scope)
	if scope == "" {
		return domain.TagScopeAll, nil
	}
	switch domain.TagScope(scope) {
	case domain.TagScopeAll, domain.TagScopeDocument, domain.TagScopeImage:
		return domain.TagScope(scope), nil
	default:
		return "", xerr.BadRequest("标签 scope 无效")
	}
}

func trimTagField(raw *string, emptyMessage string) (string, error) {
	value := strings.TrimSpace(*raw)
	if value == "" {
		return "", xerr.BadRequest(emptyMessage)
	}
	return value, nil
}

func tagIDs(rows []*models.Tag) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if row != nil {
			out = append(out, row.ID)
		}
	}
	return out
}

func loadTagCounts(ctx context.Context, repo repository.TagRepository, userID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]int64, map[uuid.UUID]int64, error) {
	docCounts, err := repo.CountDocumentsByTags(ctx, userID, ids)
	if err != nil {
		return nil, nil, err
	}
	imageCounts, err := repo.CountImagesByTags(ctx, userID, ids)
	if err != nil {
		return nil, nil, err
	}
	return docCounts, imageCounts, nil
}

// syncDocumentChunkTagsBestEffort 尽力而为的同步文档标签到 ES
func syncDocumentChunkTagsBestEffort(ctx context.Context, svcCtx *svc.ServiceContext, log *slog.Logger, userID uuid.UUID, tagID uuid.UUID, documentIDs []uuid.UUID, operation string) {
	// TODO: image tag 的 ES 同步等待图片检索索引或图片 chunk 仓储确定后接入。
	if len(documentIDs) == 0 {
		return
	}
	if svcCtx == nil || svcCtx.TagRepo == nil || svcCtx.RAGChunkRepo == nil {
		if log != nil {
			log.WarnContext(ctx, "跳过同步文档标签到 ES",
				slog.String("user_id", userID.String()),
				slog.String("tag_id", tagID.String()),
				slog.String("operation", operation),
				slog.Int("document_count", len(documentIDs)),
			)
		}
		return
	}
	tagNames, err := svcCtx.TagRepo.ListDocumentTagNames(ctx, userID, documentIDs)
	if err != nil {
		if log != nil {
			log.WarnContext(ctx, "查询文档标签名称失败，跳过同步 ES",
				slog.String("user_id", userID.String()),
				slog.String("tag_id", tagID.String()),
				slog.String("operation", operation),
				slog.Int("document_count", len(documentIDs)),
				slog.Any("error", err),
			)
		}
		return
	}
	failed := 0
	for _, documentID := range documentIDs {
		names := tagNames[documentID]
		if names == nil {
			names = []string{}
		}
		if err := svcCtx.RAGChunkRepo.UpdateTags(ctx, userID, documentID, names); err != nil {
			failed++
			if log != nil {
				log.WarnContext(ctx, "同步文档标签到 ES 失败",
					slog.String("user_id", userID.String()),
					slog.String("tag_id", tagID.String()),
					slog.String("document_id", documentID.String()),
					slog.String("operation", operation),
					slog.Any("error", err),
				)
			}
		}
	}
	if log != nil {
		log.InfoContext(ctx, "同步文档标签到 ES 完成",
			slog.String("user_id", userID.String()),
			slog.String("tag_id", tagID.String()),
			slog.String("operation", operation),
			slog.Int("document_count", len(documentIDs)),
			slog.Int("failed_count", failed),
		)
	}
}
