package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/boxify/api-go/internal/models"
	repositorypostgres "github.com/boxify/api-go/internal/repository/postgres"
	"github.com/google/uuid"
)

// 验证 ListByConversationID 会按用户和会话过滤消息，并按创建时间升序返回。
func TestMessageRepositoryListByConversationIDWhenPostgresEnvIsConfigured(t *testing.T) {
	db := newAuthTestDB(t)
	ctx := context.Background()
	userRepo := repositorypostgres.NewUserRepository(db)
	conversationRepo := repositorypostgres.NewConversationRepository(db)
	messageRepo := repositorypostgres.NewMessageRepository(db)

	userA, err := userRepo.Create(ctx, &models.User{
		Username:     "message-history-a-" + uuid.NewString(),
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("Create userA error = %v", err)
	}
	userB, err := userRepo.Create(ctx, &models.User{
		Username:     "message-history-b-" + uuid.NewString(),
		PasswordHash: "hash",
	})
	if err != nil {
		t.Fatalf("Create userB error = %v", err)
	}
	t.Cleanup(func() {
		db.WithContext(context.Background()).Exec("DELETE FROM conversations WHERE user_id IN ?", []uuid.UUID{userA.ID, userB.ID})
		db.WithContext(context.Background()).Exec("DELETE FROM users WHERE id IN ?", []uuid.UUID{userA.ID, userB.ID})
	})

	convA, err := conversationRepo.Create(ctx, userA.ID, &models.Conversation{Title: "a"})
	if err != nil {
		t.Fatalf("Create convA error = %v", err)
	}
	convOther, err := conversationRepo.Create(ctx, userA.ID, &models.Conversation{Title: "other"})
	if err != nil {
		t.Fatalf("Create convOther error = %v", err)
	}
	convB, err := conversationRepo.Create(ctx, userB.ID, &models.Conversation{Title: "b"})
	if err != nil {
		t.Fatalf("Create convB error = %v", err)
	}

	base := time.Now().Add(-time.Hour)
	rows := []*models.Message{
		{ConversationID: convA.ID, Role: "assistant", Content: "second", CreatedAt: base.Add(2 * time.Minute)},
		{ConversationID: convOther.ID, Role: "user", Content: "other conversation", CreatedAt: base.Add(time.Minute)},
		{ConversationID: convB.ID, Role: "user", Content: "other user", CreatedAt: base.Add(time.Minute)},
		{ConversationID: convA.ID, Role: "user", Content: "first", CreatedAt: base},
	}
	for _, row := range rows {
		if _, err := messageRepo.Create(ctx, rowOwner(row, userA.ID, userB.ID, convB.ID), row); err != nil {
			t.Fatalf("Create message %q error = %v", row.Content, err)
		}
	}

	got, err := messageRepo.ListByConversationID(ctx, userA.ID, convA.ID)
	if err != nil {
		t.Fatalf("ListByConversationID error = %v, want nil", err)
	}
	if len(got) != 2 {
		t.Fatalf("ListByConversationID len = %d, want 2: %#v", len(got), got)
	}
	if got[0].Content != "first" || got[1].Content != "second" {
		t.Fatalf("ListByConversationID order/content = %q,%q; want first,second", got[0].Content, got[1].Content)
	}
}

func rowOwner(row *models.Message, userA uuid.UUID, userB uuid.UUID, convB uuid.UUID) uuid.UUID {
	if row.ConversationID == convB {
		return userB
	}
	return userA
}
