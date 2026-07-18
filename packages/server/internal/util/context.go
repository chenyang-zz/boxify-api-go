package util

import (
	"context"

	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

type userIDContextKey struct{}
type knowledgeBaseIDsContextKey struct{}

// WithUserID 将已认证用户 ID 写入标准 context，供 logic/service 读取。
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	if ctx == nil || userID == uuid.Nil {
		return ctx
	}
	return context.WithValue(ctx, userIDContextKey{}, userID)
}

// UserIDFromContext 从标准 context 中读取已认证用户 ID，缺失时返回未登录错误。
func UserIDFromContext(ctx context.Context) (uuid.UUID, error) {
	if ctx == nil {
		return uuid.Nil, xerr.Unauthorized("请先登录")
	}
	userID, ok := ctx.Value(userIDContextKey{}).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return uuid.Nil, xerr.Unauthorized("请先登录")
	}
	return userID, nil
}

// WithKnowledgeBaseIDs 将可信知识库范围写入标准 context，供工具和 service 读取。
//
// ctx 为 nil 或 kbIDs 过滤后为空时返回原 ctx。uuid.Nil 会被忽略，重复 ID 会按首次出现顺序去重。
func WithKnowledgeBaseIDs(ctx context.Context, kbIDs []uuid.UUID) context.Context {
	if ctx == nil {
		return ctx
	}
	cleaned := cleanKnowledgeBaseIDs(kbIDs)
	if len(cleaned) == 0 {
		return ctx
	}
	return context.WithValue(ctx, knowledgeBaseIDsContextKey{}, cleaned)
}

// KnowledgeBaseIDsFromContext 从标准 context 中读取可信知识库范围。
//
// 缺少范围或范围为空时返回 bad request 错误；返回的切片是副本，调用方修改不会影响 context 中的值。
func KnowledgeBaseIDsFromContext(ctx context.Context) ([]uuid.UUID, error) {
	if ctx == nil {
		return nil, xerr.BadRequest("知识库范围不能为空")
	}
	kbIDs, ok := ctx.Value(knowledgeBaseIDsContextKey{}).([]uuid.UUID)
	if !ok {
		return nil, xerr.BadRequest("知识库范围不能为空")
	}
	cleaned := cleanKnowledgeBaseIDs(kbIDs)
	if len(cleaned) == 0 {
		return nil, xerr.BadRequest("知识库范围不能为空")
	}
	return cleaned, nil
}

func cleanKnowledgeBaseIDs(kbIDs []uuid.UUID) []uuid.UUID {
	if len(kbIDs) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(kbIDs))
	out := make([]uuid.UUID, 0, len(kbIDs))
	for _, kbID := range kbIDs {
		if kbID == uuid.Nil {
			continue
		}
		if _, ok := seen[kbID]; ok {
			continue
		}
		seen[kbID] = struct{}{}
		out = append(out, kbID)
	}
	return out
}
