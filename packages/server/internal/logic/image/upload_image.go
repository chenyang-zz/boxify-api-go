package image

import (
	"context"
	"log/slog"

	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/storage"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util/uploadfile"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// UploadImageLogic contains the uploadImage use case.
type UploadImageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewUploadImageLogic creates a UploadImageLogic.
func NewUploadImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadImageLogic {
	return &UploadImageLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.image.uploadimage"),
	}
}

// UploadImage 上传图片
func (l *UploadImageLogic) UploadImage(userID uuid.UUID, input *request.UploadImageRequest) (*response.ImageResponse, error) {
	if input == nil || input.File == nil {
		return nil, xerr.BadRequest("上传文件不能为空")
	}
	fileInfo, err := uploadfile.Read(input.File, maxImageFileSize, "文件超过 20MB 限制", "读取上传文件失败")
	if err != nil {
		return nil, err
	}
	if fileInfo.FileName == "" {
		return nil, xerr.BadRequest("文件名不能为空")
	}
	ext, err := supportedImageExt(fileInfo.Ext)
	if err != nil {
		return nil, err
	}
	kbID, err := resolveImageKnowledgeBaseID(l.ctx, l.svcCtx.KnowledgeBaseRepo, l.log, userID, input.KBID)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.Storage == nil {
		return nil, xerr.BadRequest("对象存储未初始化")
	}

	imageID := uuid.New()
	fileKey := storage.BuildFileKey(userID, "images", imageID, ext)
	if err := l.svcCtx.Storage.Put(l.ctx, fileKey, fileInfo.Content); err != nil {
		return nil, err
	}

	row, err := l.svcCtx.ImageRepo.Create(l.ctx, userID, &models.Image{
		ID:       imageID,
		KBID:     &kbID,
		FileName: fileInfo.FileName,
		FileExt:  ext,
		FileSize: fileInfo.Size,
		FileKey:  fileKey,
		Status:   types.ImageStatusPending,
	})
	if err != nil {
		return nil, err
	}
	l.log.InfoContext(l.ctx, "图片上传成功",
		slog.String("user_id", userID.String()),
		slog.String("image_id", row.ID.String()),
		slog.String("kb_id", kbID.String()),
		slog.String("file_ext", ext),
		slog.Int64("file_size", row.FileSize),
	)

	if err := enqueueParseImageTask(l.ctx, l.svcCtx.TaskProducer, userID, row.ID); err != nil {
		markImageParseDispatchFailed(l.ctx, l.svcCtx.ImageRepo, userID, row.ID, err)
		return nil, err
	}
	l.log.InfoContext(l.ctx, "图片解析任务已入队",
		slog.String("image_id", row.ID.String()),
	)
	return mapper.ImageToResponse(row, nil, imageURL(l.svcCtx, row.FileKey)), nil
}
