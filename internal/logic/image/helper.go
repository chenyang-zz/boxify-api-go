package image

import (
	"context"
	"log/slog"
	"strings"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/queue"
	knowledgebaselogic "github.com/boxify/api-go/internal/logic/knowledgebase"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const maxImageFileSize = 20 * 1024 * 1024

var supportedImageExts = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".webp": {},
	".bmp":  {},
}

func parseImageID(raw string) (uuid.UUID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return uuid.Nil, xerr.BadRequest("图片 ID 无效")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("图片 ID 无效")
	}
	return id, nil
}

func parseOptionalKBID(raw *string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, xerr.BadRequest("知识库 ID 无效")
	}
	return &id, nil
}

func parseRequiredKBID(raw string) (uuid.UUID, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return uuid.Nil, xerr.BadRequest("知识库 ID 无效")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("知识库 ID 无效")
	}
	return id, nil
}

func supportedImageExt(ext string) (string, error) {
	ext = strings.ToLower(ext)
	if _, ok := supportedImageExts[ext]; !ok {
		return "", xerr.BadRequestf("不支持的文件类型: %s", ext)
	}
	return ext, nil
}

// resolveImageKnowledgeBaseID 解析图片知识库 ID，如果未指定则确保用户有默认知识库并返回其 ID。
func resolveImageKnowledgeBaseID(ctx context.Context, repo repository.KnowledgeBaseRepository, log *slog.Logger, userID uuid.UUID, rawKBID *string) (uuid.UUID, error) {
	if repo == nil {
		return uuid.Nil, xerr.BadRequest("知识库仓储未初始化")
	}
	if parsed, err := parseOptionalKBID(rawKBID); err != nil {
		return uuid.Nil, err
	} else if parsed != nil {
		row, err := repo.FindByID(ctx, userID, *parsed)
		if err != nil {
			return uuid.Nil, err
		}
		return row.ID, nil
	}
	row, _, err := knowledgebaselogic.EnsureDefaultKnowledgeBase(ctx, repo, userID, log)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}

// 队列提交图片解析任务
func enqueueParseImageTask(ctx context.Context, producer queue.Producer, userID uuid.UUID, imageID uuid.UUID) error {
	if producer == nil {
		return xerr.Internal("任务队列未初始化", nil)
	}
	task, err := types.NewParseImageTask(userID, imageID)
	if err != nil {
		return xerr.Wrapf(err, "创建图片解析任务失败")
	}
	_, err = producer.Enqueue(ctx, task)
	if err != nil {
		return xerr.Wrapf(err, "提交图片解析任务失败")
	}
	return nil
}

// 标记图片解析任务分发失败
func markImageParseDispatchFailed(ctx context.Context, repo repository.ImageRepository, userID uuid.UUID, imageID uuid.UUID, cause error) {
	if repo == nil || cause == nil {
		return
	}
	message := cause.Error()
	_, _ = repo.UpdateFields(ctx, userID, imageID, &models.Image{
		Status:   types.ImageStatusFailed,
		ErrorMsg: &message,
	}, repository.NewImageUpdateFields().Status().ErrorMsg())
}

// 最努力地删除图片检索 chunk
func deleteImageChunksBestEffort(ctx context.Context, svcCtx *svc.ServiceContext, log *slog.Logger, userID uuid.UUID, imageID uuid.UUID) {
	if svcCtx == nil || svcCtx.RAGChunkRepo == nil {
		return
	}
	if err := svcCtx.RAGChunkRepo.DeleteByDocument(ctx, userID, imageID); err != nil && log != nil {
		log.WarnContext(ctx, "清理图片检索 chunk 失败（忽略）",
			slog.String("user_id", userID.String()),
			slog.String("image_id", imageID.String()),
			slog.String("error", err.Error()),
		)
	}
}

// 最努力地更新图片检索 chunk 的知识库归属
func updateImageChunksKnowledgeBaseBestEffort(ctx context.Context, svcCtx *svc.ServiceContext, log *slog.Logger, userID uuid.UUID, imageID uuid.UUID, kbID uuid.UUID) {
	if svcCtx == nil || svcCtx.RAGChunkRepo == nil {
		return
	}
	if err := svcCtx.RAGChunkRepo.UpdateKnowledgeBase(ctx, userID, imageID, kbID); err != nil && log != nil {
		log.WarnContext(ctx, "更新图片检索 chunk 知识库归属失败（忽略）",
			slog.String("user_id", userID.String()),
			slog.String("image_id", imageID.String()),
			slog.String("kb_id", kbID.String()),
			slog.String("error", err.Error()),
		)
	}
}

// 生成图片访问 URL
func imageURL(svcCtx *svc.ServiceContext, fileKey string) string {
	if svcCtx == nil || svcCtx.URLSigner == nil || strings.TrimSpace(fileKey) == "" {
		return ""
	}
	return svcCtx.URLSigner.URL(fileKey)
}
