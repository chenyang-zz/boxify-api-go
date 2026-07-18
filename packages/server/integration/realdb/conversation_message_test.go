package realdb_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	dbpostgres "github.com/boxify/api-go/internal/infrastructure/db/postgres"
	"github.com/boxify/api-go/internal/models"
	repositorypostgres "github.com/boxify/api-go/internal/repository/postgres"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type authData struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	AccessToken string    `json:"access_token"`
}

type conversationData struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
}

type conversationPageData struct {
	Total int64               `json:"total"`
	List  []*conversationData `json:"list"`
}

type messageData struct {
	ID      uuid.UUID `json:"id"`
	Role    string    `json:"role"`
	Content string    `json:"content"`
}

type messagePageData struct {
	List    []*messageData `json:"list"`
	HasMore bool           `json:"has_more"`
}

// TestConversationAndMessagePersistenceAndUserIsolation 验证真实 API 与 PostgreSQL 下的会话持久化、消息分页、跨用户隔离和级联删除。
func TestConversationAndMessagePersistenceAndUserIsolation(t *testing.T) {
	apiURL := strings.TrimRight(os.Getenv("COVE_REAL_DB_API_URL"), "/")
	databaseURL := os.Getenv("COVE_REAL_DB_DATABASE_URL")
	if apiURL == "" || databaseURL == "" {
		t.Skip("COVE_REAL_DB_API_URL and COVE_REAL_DB_DATABASE_URL are required")
	}

	ctx := t.Context()
	db, err := dbpostgres.NewGormDB(ctx, dbpostgres.Config{URL: databaseURL})
	if err != nil {
		t.Fatalf("NewGormDB error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("DB error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := sqlDB.Close(); closeErr != nil {
			t.Errorf("Close DB error = %v", closeErr)
		}
	})

	client := &http.Client{Timeout: 10 * time.Second}
	runID := os.Getenv("COVE_REAL_DB_RUN_ID")
	owner := registerUser(t, client, apiURL, testUsername("owner", runID))
	other := registerUser(t, client, apiURL, testUsername("other", runID))
	t.Cleanup(func() {
		result := db.WithContext(context.Background()).Where("id IN ?", []uuid.UUID{owner.UserID, other.UserID}).Delete(&models.User{})
		if result.Error != nil {
			t.Errorf("cleanup users error = %v", result.Error)
		}
	})

	ownerConversation := createConversation(t, client, apiURL, owner.AccessToken, "owner-private")
	otherConversation := createConversation(t, client, apiURL, other.AccessToken, "other-private")
	assertConversationList(t, client, apiURL, owner.AccessToken, ownerConversation.ID, "owner-private")
	assertConversationList(t, client, apiURL, other.AccessToken, otherConversation.ID, "other-private")

	messageRepo := repositorypostgres.NewMessageRepository(db)
	baseTime := time.Now().UTC().Add(-time.Hour).Truncate(time.Millisecond)
	seededMessages := []*models.Message{
		{ConversationID: ownerConversation.ID, Role: "user", Content: "first persisted message", CreatedAt: baseTime},
		{ConversationID: ownerConversation.ID, Role: "assistant", Content: "second persisted message", CreatedAt: baseTime.Add(time.Minute)},
		{ConversationID: ownerConversation.ID, Role: "assistant", Content: "third persisted message", CreatedAt: baseTime.Add(2 * time.Minute)},
	}
	for _, message := range seededMessages {
		if _, createErr := messageRepo.Create(ctx, owner.UserID, message); createErr != nil {
			t.Fatalf("MessageRepository.Create(%q) error = %v", message.Content, createErr)
		}
	}

	latest := doJSON[messagePageData](
		t,
		client,
		http.MethodGet,
		fmt.Sprintf("%s/api/conversation/%s/messages?limit=2", apiURL, ownerConversation.ID),
		owner.AccessToken,
		nil,
		http.StatusOK,
	)
	if latest.Code != 0 || !latest.Data.HasMore || len(latest.Data.List) != 2 {
		t.Fatalf("latest message page = %+v, want code 0, has_more true, and 2 rows", latest)
	}
	if latest.Data.List[0].Content != "second persisted message" || latest.Data.List[1].Content != "third persisted message" {
		t.Fatalf("latest message contents = %q, %q; want second and third", latest.Data.List[0].Content, latest.Data.List[1].Content)
	}

	earlier := doJSON[messagePageData](
		t,
		client,
		http.MethodGet,
		fmt.Sprintf("%s/api/conversation/%s/messages?limit=2&before=%s", apiURL, ownerConversation.ID, latest.Data.List[0].ID),
		owner.AccessToken,
		nil,
		http.StatusOK,
	)
	if earlier.Code != 0 || earlier.Data.HasMore || len(earlier.Data.List) != 1 || earlier.Data.List[0].Content != "first persisted message" {
		t.Fatalf("earlier message page = %+v, want only first persisted message", earlier)
	}

	deniedMessages := doJSON[json.RawMessage](
		t,
		client,
		http.MethodGet,
		fmt.Sprintf("%s/api/conversation/%s/messages", apiURL, ownerConversation.ID),
		other.AccessToken,
		nil,
		http.StatusNotFound,
	)
	if deniedMessages.Code != 40400 {
		t.Fatalf("cross-user message list code = %d, want 40400", deniedMessages.Code)
	}

	deniedRename := doJSON[json.RawMessage](
		t,
		client,
		http.MethodPatch,
		fmt.Sprintf("%s/api/conversation/%s", apiURL, ownerConversation.ID),
		other.AccessToken,
		map[string]string{"title": "forbidden rename"},
		http.StatusNotFound,
	)
	if deniedRename.Code != 40400 {
		t.Fatalf("cross-user rename code = %d, want 40400", deniedRename.Code)
	}

	deniedDelete := doJSON[json.RawMessage](
		t,
		client,
		http.MethodDelete,
		fmt.Sprintf("%s/api/conversation/%s", apiURL, ownerConversation.ID),
		other.AccessToken,
		nil,
		http.StatusNotFound,
	)
	if deniedDelete.Code != 40400 {
		t.Fatalf("cross-user delete code = %d, want 40400", deniedDelete.Code)
	}

	renamed := doJSON[conversationData](
		t,
		client,
		http.MethodPatch,
		fmt.Sprintf("%s/api/conversation/%s", apiURL, ownerConversation.ID),
		owner.AccessToken,
		map[string]string{"title": "owner-renamed"},
		http.StatusOK,
	)
	if renamed.Code != 0 || renamed.Data.Title != "owner-renamed" {
		t.Fatalf("owner rename response = %+v, want owner-renamed", renamed)
	}
	assertConversationList(t, client, apiURL, owner.AccessToken, ownerConversation.ID, "owner-renamed")

	var persistedConversation models.Conversation
	if findErr := db.WithContext(ctx).Where("id = ? AND user_id = ?", ownerConversation.ID, owner.UserID).First(&persistedConversation).Error; findErr != nil {
		t.Fatalf("query persisted conversation error = %v", findErr)
	}
	if persistedConversation.Title != "owner-renamed" {
		t.Fatalf("persisted conversation title = %q, want owner-renamed", persistedConversation.Title)
	}
	var persistedMessageCount int64
	if countErr := db.WithContext(ctx).Model(&models.Message{}).Where("conversation_id = ?", ownerConversation.ID).Count(&persistedMessageCount).Error; countErr != nil {
		t.Fatalf("count persisted messages error = %v", countErr)
	}
	if persistedMessageCount != int64(len(seededMessages)) {
		t.Fatalf("persisted message count = %d, want %d", persistedMessageCount, len(seededMessages))
	}

	deleted := doJSON[json.RawMessage](
		t,
		client,
		http.MethodDelete,
		fmt.Sprintf("%s/api/conversation/%s", apiURL, ownerConversation.ID),
		owner.AccessToken,
		nil,
		http.StatusOK,
	)
	if deleted.Code != 0 {
		t.Fatalf("owner delete code = %d, want 0", deleted.Code)
	}
	assertDeletedFromDatabase(t, db, ownerConversation.ID, otherConversation.ID)

	doJSON[json.RawMessage](
		t,
		client,
		http.MethodDelete,
		fmt.Sprintf("%s/api/conversation/%s", apiURL, otherConversation.ID),
		other.AccessToken,
		nil,
		http.StatusOK,
	)
}

func registerUser(t *testing.T, client *http.Client, apiURL string, username string) *authData {
	t.Helper()
	response := doJSON[authData](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/auth/register",
		"",
		map[string]string{
			"username": username,
			"email":    username + "@example.test",
			"password": "Cove-realdb-123!",
		},
		http.StatusOK,
	)
	if response.Code != 0 || response.Data.UserID == uuid.Nil || response.Data.AccessToken == "" {
		t.Fatalf(
			"register response for %q has code=%d user_id=%s username=%q access_token_present=%v, want authenticated user",
			username,
			response.Code,
			response.Data.UserID,
			response.Data.Username,
			response.Data.AccessToken != "",
		)
	}
	return &response.Data
}

func testUsername(label string, runID string) string {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		runID = "local"
	}
	if len(runID) > 20 {
		runID = runID[len(runID)-20:]
	}
	return fmt.Sprintf("realdb-%s-%s-%s", label, runID, uuid.NewString()[:8])
}

func createConversation(t *testing.T, client *http.Client, apiURL string, accessToken string, title string) *conversationData {
	t.Helper()
	response := doJSON[conversationData](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/conversation/",
		accessToken,
		map[string]string{"title": title},
		http.StatusOK,
	)
	if response.Code != 0 || response.Data.ID == uuid.Nil || response.Data.Title != title {
		t.Fatalf("create conversation response = %+v, want title %q", response, title)
	}
	return &response.Data
}

func assertConversationList(t *testing.T, client *http.Client, apiURL string, accessToken string, conversationID uuid.UUID, title string) {
	t.Helper()
	response := doJSON[conversationPageData](
		t,
		client,
		http.MethodGet,
		apiURL+"/api/conversation/?page=1&page_size=20",
		accessToken,
		nil,
		http.StatusOK,
	)
	if response.Code != 0 || response.Data.Total != 1 || len(response.Data.List) != 1 {
		t.Fatalf("conversation list = %+v, want one user-scoped row", response)
	}
	if response.Data.List[0].ID != conversationID || response.Data.List[0].Title != title {
		t.Fatalf("conversation list row = %+v, want id %s and title %q", response.Data.List[0], conversationID, title)
	}
}

func assertDeletedFromDatabase(t *testing.T, db *gorm.DB, deletedConversationID uuid.UUID, survivingConversationID uuid.UUID) {
	t.Helper()
	ctx := t.Context()
	var deletedConversationCount int64
	if err := db.WithContext(ctx).Model(&models.Conversation{}).Where("id = ?", deletedConversationID).Count(&deletedConversationCount).Error; err != nil {
		t.Fatalf("count deleted conversation error = %v", err)
	}
	if deletedConversationCount != 0 {
		t.Fatalf("deleted conversation count = %d, want 0", deletedConversationCount)
	}
	var deletedMessageCount int64
	if err := db.WithContext(ctx).Model(&models.Message{}).Where("conversation_id = ?", deletedConversationID).Count(&deletedMessageCount).Error; err != nil {
		t.Fatalf("count cascade-deleted messages error = %v", err)
	}
	if deletedMessageCount != 0 {
		t.Fatalf("cascade-deleted message count = %d, want 0", deletedMessageCount)
	}
	var survivingConversationCount int64
	if err := db.WithContext(ctx).Model(&models.Conversation{}).Where("id = ?", survivingConversationID).Count(&survivingConversationCount).Error; err != nil {
		t.Fatalf("count surviving conversation error = %v", err)
	}
	if survivingConversationCount != 1 {
		t.Fatalf("other user's surviving conversation count = %d, want 1", survivingConversationCount)
	}
}

func doJSON[T any](t *testing.T, client *http.Client, method string, requestURL string, accessToken string, body any, wantStatus int) *envelope[T] {
	t.Helper()
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal(%s %s) error = %v", method, requestURL, err)
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(t.Context(), method, requestURL, requestBody)
	if err != nil {
		t.Fatalf("NewRequestWithContext(%s %s) error = %v", method, requestURL, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		request.Header.Set("Authorization", "Bearer "+accessToken)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("HTTP %s %s error = %v", method, requestURL, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll(%s %s) error = %v", method, requestURL, err)
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("HTTP %s %s status = %d, want %d; response_bytes=%d", method, requestURL, response.StatusCode, wantStatus, len(responseBody))
	}
	var decoded envelope[T]
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		t.Fatalf("Unmarshal(%s %s) error = %v; response_bytes=%d", method, requestURL, err, len(responseBody))
	}
	return &decoded
}
