package skill

import (
	"context"
	"log/slog"

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
