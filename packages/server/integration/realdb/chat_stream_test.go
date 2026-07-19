package realdb_test

import (
	"bufio"
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
	"github.com/google/uuid"
)

type modelConfigData struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Provider  string    `json:"provider"`
	ModelName string    `json:"model_name"`
	BaseURL   string    `json:"base_url"`
	IsDefault bool      `json:"is_default"`
}

type chatSSEEvent struct {
	Type           string    `json:"type"`
	Text           string    `json:"text"`
	Content        string    `json:"content"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
}

type chatStreamResult struct {
	ConversationID uuid.UUID
	Title          string
	Answer         string
	AssistantID    uuid.UUID
	SawThinking    bool
	SawThinkDone   bool
	SawDone        bool
}

// TestChatStreamCreatesAndPersistsMessages 验证公共 Chat/SSE 链路会通过 PostgreSQL、Redis
// 与本地确定性 OpenAI 兼容服务生成并持久化消息，且不通过仓储夹具预置消息。
func TestChatStreamCreatesAndPersistsMessages(t *testing.T) {
	apiURL := strings.TrimRight(os.Getenv("COVE_REAL_DB_API_URL"), "/")
	databaseURL := os.Getenv("COVE_REAL_DB_DATABASE_URL")
	providerURL := strings.TrimRight(os.Getenv("COVE_REAL_DB_LLM_URL"), "/")
	expectedAnswer := strings.TrimSpace(os.Getenv("COVE_REAL_DB_LLM_ANSWER"))
	if apiURL == "" || databaseURL == "" || providerURL == "" || expectedAnswer == "" {
		t.Skip("COVE_REAL_DB_API_URL, COVE_REAL_DB_DATABASE_URL, COVE_REAL_DB_LLM_URL, and COVE_REAL_DB_LLM_ANSWER are required")
	}

	db, err := dbpostgres.NewGormDB(t.Context(), dbpostgres.Config{URL: databaseURL})
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

	client := &http.Client{Timeout: 20 * time.Second}
	runID := os.Getenv("COVE_REAL_DB_RUN_ID")
	owner := registerUser(t, client, apiURL, testUsername("chat-stream", runID))
	t.Cleanup(func() {
		result := db.WithContext(context.Background()).Where("id = ?", owner.UserID).Delete(&models.User{})
		if result.Error != nil {
			t.Errorf("cleanup chat stream user error = %v", result.Error)
		}
	})

	configured := doJSON[modelConfigData](
		t,
		client,
		http.MethodPost,
		apiURL+"/api/model-configs/",
		owner.AccessToken,
		map[string]any{
			"type":       "chat",
			"provider":   "openai",
			"name":       "Local deterministic chat",
			"model_name": "cove-e2e-chat",
			"api_key":    "cove-e2e-local-key",
			"base_url":   providerURL,
			"is_default": true,
		},
		http.StatusOK,
	)
	if configured.Code != 0 || configured.Data.ID == uuid.Nil || !configured.Data.IsDefault || configured.Data.BaseURL != providerURL {
		t.Fatalf("model config response = %+v, want a default local provider", configured)
	}

	prompt := "persist-chat-" + uuid.NewString()[:6]
	streamed := streamChatThroughAPI(t, client, apiURL, owner.AccessToken, prompt)
	if streamed.ConversationID == uuid.Nil || streamed.Title != prompt {
		t.Fatalf("stream meta conversation_id=%s title=%q, want non-zero ID and %q", streamed.ConversationID, streamed.Title, prompt)
	}
	if streamed.Answer != expectedAnswer {
		t.Fatalf("stream answer = %q, want %q", streamed.Answer, expectedAnswer)
	}
	if !streamed.SawThinking || !streamed.SawThinkDone || !streamed.SawDone || streamed.AssistantID == uuid.Nil {
		t.Fatalf(
			"stream terminal state thinking=%v think_done=%v done=%v assistant_id=%s, want a complete lifecycle",
			streamed.SawThinking,
			streamed.SawThinkDone,
			streamed.SawDone,
			streamed.AssistantID,
		)
	}

	assertConversationList(t, client, apiURL, owner.AccessToken, streamed.ConversationID, prompt)
	history := doJSON[messagePageData](
		t,
		client,
		http.MethodGet,
		fmt.Sprintf("%s/api/conversation/%s/messages?limit=10", apiURL, streamed.ConversationID),
		owner.AccessToken,
		nil,
		http.StatusOK,
	)
	if history.Code != 0 || history.Data.HasMore || len(history.Data.List) != 2 {
		t.Fatalf("chat history = %+v, want exactly two generated messages", history)
	}
	if history.Data.List[0].Role != "user" || history.Data.List[0].Content != prompt {
		t.Fatalf("persisted user message = %+v, want %q", history.Data.List[0], prompt)
	}
	if history.Data.List[1].ID != streamed.AssistantID || history.Data.List[1].Role != "assistant" || history.Data.List[1].Content != expectedAnswer {
		t.Fatalf("persisted assistant message = %+v, want id %s and %q", history.Data.List[1], streamed.AssistantID, expectedAnswer)
	}

	var persisted []models.Message
	if err := db.WithContext(t.Context()).Where("conversation_id = ?", streamed.ConversationID).Order("created_at ASC").Find(&persisted).Error; err != nil {
		t.Fatalf("query persisted generated messages error = %v", err)
	}
	if len(persisted) != 2 || persisted[0].Content != prompt || persisted[1].Content != expectedAnswer {
		t.Fatalf("database generated messages = %+v, want user prompt and deterministic assistant answer", persisted)
	}
}

func streamChatThroughAPI(t *testing.T, client *http.Client, apiURL string, accessToken string, prompt string) chatStreamResult {
	t.Helper()
	requestBody, err := json.Marshal(map[string]any{
		"message":          prompt,
		"enable_knowledge": false,
	})
	if err != nil {
		t.Fatalf("marshal chat stream request error = %v", err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, apiURL+"/api/chat/stream", bytes.NewReader(requestBody))
	if err != nil {
		t.Fatalf("create chat stream request error = %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("chat stream request error = %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("chat stream status = %d, want 200; body=%q", response.StatusCode, body)
	}
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("chat stream Content-Type = %q, want text/event-stream", contentType)
	}

	result := chatStreamResult{}
	reader := bufio.NewReader(response.Body)
	eventName := ""
	dataLines := make([]string, 0, 1)
	consume := func() {
		if eventName == "" || len(dataLines) == 0 {
			return
		}
		var event chatSSEEvent
		if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &event); err != nil {
			t.Fatalf("decode SSE event %q error = %v", eventName, err)
		}
		switch eventName {
		case "meta":
			result.ConversationID = event.ConversationID
			result.Title = event.Title
		case "think":
			result.SawThinking = result.SawThinking || event.Status == "thinking"
			result.SawThinkDone = result.SawThinkDone || event.Status == "done"
		case "token":
			result.Answer += event.Text
		case "done":
			result.SawDone = true
			result.AssistantID, err = uuid.Parse(event.Text)
			if err != nil {
				t.Fatalf("parse done assistant ID %q error = %v", event.Text, err)
			}
		case "error":
			t.Fatalf("chat stream returned error event: %s", event.Content)
		}
	}

	for {
		line, readErr := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			consume()
			eventName = ""
			dataLines = dataLines[:0]
		} else if strings.HasPrefix(line, "event:") {
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
		if readErr != nil {
			if readErr != io.EOF {
				t.Fatalf("read chat SSE error = %v", readErr)
			}
			consume()
			break
		}
	}
	return result
}
