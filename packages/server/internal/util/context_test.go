package util_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

func TestUserIDFromContext(t *testing.T) {
	userID := uuid.New()
	ctx := util.WithUserID(context.Background(), userID)

	got, err := util.UserIDFromContext(ctx)
	if err != nil {
		t.Fatalf("UserIDFromContext error = %v, want nil", err)
	}
	if got != userID {
		t.Fatalf("UserIDFromContext = %s, want %s", got, userID)
	}
}

func TestUserIDFromContextReturnsUnauthorizedWhenMissingOrNil(t *testing.T) {
	if got, err := util.UserIDFromContext(context.Background()); got != uuid.Nil || xerr.From(err).Kind != xerr.KindUnauthorized {
		t.Fatalf("missing UserIDFromContext = %s, %v; want nil,unauthorized", got, err)
	}
	if got, err := util.UserIDFromContext(util.WithUserID(context.Background(), uuid.Nil)); got != uuid.Nil || xerr.From(err).Kind != xerr.KindUnauthorized {
		t.Fatalf("nil UserIDFromContext = %s, %v; want nil,unauthorized", got, err)
	}
}

// 验证知识库 ID 会写入 context，并在写入时过滤空值和去重。
func TestKnowledgeBaseIDsFromContextFiltersNilAndDuplicates(t *testing.T) {
	first := uuid.New()
	second := uuid.New()
	ctx := util.WithKnowledgeBaseIDs(context.Background(), []uuid.UUID{uuid.Nil, first, second, first})

	got, err := util.KnowledgeBaseIDsFromContext(ctx)
	if err != nil {
		t.Fatalf("KnowledgeBaseIDsFromContext error = %v, want nil", err)
	}
	want := []uuid.UUID{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("KnowledgeBaseIDsFromContext = %#v, want %#v", got, want)
	}
}

// 验证缺少知识库范围或只传空 UUID 时，读取 context 会返回错误。
func TestKnowledgeBaseIDsFromContextReturnsBadRequestWhenMissingOrEmpty(t *testing.T) {
	if got, err := util.KnowledgeBaseIDsFromContext(context.Background()); got != nil || xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("missing KnowledgeBaseIDsFromContext = %#v, %v; want nil,bad_request", got, err)
	}
	if got, err := util.KnowledgeBaseIDsFromContext(util.WithKnowledgeBaseIDs(context.Background(), []uuid.UUID{uuid.Nil})); got != nil || xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("empty KnowledgeBaseIDsFromContext = %#v, %v; want nil,bad_request", got, err)
	}
}
