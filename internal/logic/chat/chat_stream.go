package chat

import (
	"context"
	"strings"
	"time"

	"log/slog"

	"github.com/boxify/api-go/internal/core/llm"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/realtime"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/google/uuid"
)

// ChatStreamLogic contains the chatStream use case.
type ChatStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewChatStreamLogic creates a ChatStreamLogic.
func NewChatStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatStreamLogic {
	return &ChatStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.chat.chatstream"),
	}
}

// ChatStream 流式聊天
func (l *ChatStreamLogic) ChatStream(userID uuid.UUID, input *request.ChatStreamRequest) (<-chan types.Event, error) {

	// 生成动作不在当前协程中，而是一个独立 协程 的后台任务生成
	// 通过 Redis 频道广播 token；本协程只「订阅频道并转发」给当前客户端
	// 这样客户端中途断开（切页面/关标签）只会停止转发，后台生成照常跑完并落库——回来重拉历史能看到完整回复
	// 生成中重连还能续传（见 resume_events）

	userText := strings.TrimSpace(input.Message)

	attachments := make([]*types.MessageAttachment, 0, len(input.Attachments))
	for _, attachment := range input.Attachments {
		if attachment.Text != "" {
			attachments = append(attachments, &types.MessageAttachment{
				Content:  attachment.Text,
				FileName: attachment.FileName,
			})
		}
	}

	conversation, err := l.ensureConversation(userID, input.ConversationID, input.Message)
	if err != nil {
		return nil, err
	}

	// AI 主动开场白（今日回顾「聊聊」）：仅新会话首轮，先把开场白作为
	// assistant 消息落库，使其进入对话历史，模型回复时能接住这个话题
	greeting := strings.TrimSpace(input.Greeting)
	messageCount, err := l.svcCtx.MessageRepo.Count(l.ctx, conversation.ID)
	if len(greeting) != 0 && messageCount == 0 {
		l.log.InfoContext(l.ctx, "添加开场白", slog.String("greeting", greeting))
		_, err = l.svcCtx.MessageRepo.Create(l.ctx, userID, &models.Message{
			ConversationID: conversation.ID,
			Role:           string(llm.AssistantRole),
			Content:        greeting,
		})
		if err != nil {
			l.log.WarnContext(l.ctx, "添加开场白失败", slog.String("greeting", greeting), slog.String("error", err.Error()))
		}
	}
	_, err = l.svcCtx.MessageRepo.Create(l.ctx, userID, &models.Message{
		ConversationID: conversation.ID,
		Role:           string(llm.UserRole),
		Content:        userText,
	})
	if err != nil {
		return nil, err
	}

	topic := realtime.ConversationTopic(conversation.ID)
	subscription, err := l.svcCtx.Realtime.Subscribe(l.ctx, topic)
	if err != nil {
		return nil, err
	}

	events := make(chan types.Event, 16)
	go func() {
		select {
		case <-l.ctx.Done():
			close(events)
			_ = subscription.Close(context.Background())
			return
		case events <- types.NewMetaEvent(conversation.ID, conversation.Title):
		}
		err = realtime.Forward(l.ctx, subscription, events, realtime.ForwardOptions{})
		if err != nil {
			l.log.WarnContext(l.ctx, "转发事件失败", slog.String("error", err.Error()))
		}
	}()

	go func() {
		for {
			select {
			case <-l.ctx.Done():
				return
			default:
				time.Sleep(2 * time.Second)
				_ = l.svcCtx.Realtime.Publish(l.ctx, topic, types.NewTokenEvent("345"))
			}

		}

	}()
	return events, nil
}

func (l *ChatStreamLogic) ensureConversation(userID uuid.UUID, conversationIDStr string, message string) (*models.Conversation, error) {
	conversationID, err := parseConversationID(conversationIDStr)
	if err == nil {
		conversation, err := l.svcCtx.ConversationRepo.FindByID(l.ctx, userID, conversationID)
		if err == nil {
			return conversation, nil
		}
	}

	var title string
	if message == "" {
		title = "新对话"
	} else if len(message) <= 20 {
		title = message
	} else {
		title = message[:20]
	}

	return l.svcCtx.ConversationRepo.Create(l.ctx, userID, &models.Conversation{Title: title})
}
