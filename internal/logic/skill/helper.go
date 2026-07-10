package skill

import (
	"context"
	"log/slog"

	coretool "github.com/boxify/api-go/internal/core/tool"
	domaintools "github.com/boxify/api-go/internal/domain/tools"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const (
	defaultSkillIcon     = "🧩"
	defaultMaxSkillCount = 200
)

func skillIDFromInput(input *request.UriSkillIDRequest) (uuid.UUID, error) {
	if input == nil {
		return uuid.Nil, xerr.BadRequest("技能 ID 无效")
	}
	id, err := uuid.Parse(input.ID)
	if err != nil {
		return uuid.Nil, xerr.BadRequest("技能 ID 无效")
	}
	return id, nil
}

func resolveSkillKnowledgeBaseID(ctx context.Context, svcCtx *svc.ServiceContext, userID uuid.UUID, raw string) (*uuid.UUID, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, xerr.BadRequest("知识库 ID 无效")
	}
	if _, err := svcCtx.KnowledgeBaseRepo.FindByID(ctx, userID, id); err != nil {
		return nil, err
	}
	return &id, nil
}

// validateSkillToolKeys 规范化工具键并确认每个键均存在于当前内置工具目录中。
func validateSkillToolKeys(ctx context.Context, svcCtx *svc.ServiceContext, raw []string) (models.StringList, error) {
	toolKeys := mapper.SkillToolKeysFromRequest(raw)
	if len(toolKeys) == 0 {
		return toolKeys, nil
	}

	// 通过领域工具目录构建完整注册表，避免技能层维护重复的工具键清单。
	catalog, err := domaintools.NewCatalog(svcCtx)
	if err != nil {
		return nil, err
	}
	registry, err := catalog.BuildRegistry(ctx, coretool.Selection{})
	if err != nil {
		return nil, err
	}
	for _, toolKey := range toolKeys {
		if _, exists := registry.Lookup(toolKey); !exists {
			return nil, xerr.NotFound("工具不存在")
		}
	}
	return toolKeys, nil
}

func ensureSkillLimit(ctx context.Context, svcCtx *svc.ServiceContext, userID uuid.UUID, log *slog.Logger) error {
	maxCount := svcCtx.Config.Skill.MaxCount
	if maxCount <= 0 {
		maxCount = defaultMaxSkillCount
	}
	rows, err := svcCtx.SkillRepo.List(ctx, userID)
	if err != nil {
		return err
	}
	if len(rows) < maxCount {
		return nil
	}
	if log != nil {
		log.WarnContext(ctx, "技能数量达到上限",
			slog.String("user_id", userID.String()),
			slog.Int("current_count", len(rows)),
			slog.Int("max_count", maxCount),
		)
	}
	return xerr.BadRequest("技能数量已达到上限")
}
