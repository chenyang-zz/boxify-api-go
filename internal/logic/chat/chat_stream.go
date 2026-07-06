package chat

import (
	"context"
	"strings"

	"log/slog"

	coreagent "github.com/boxify/api-go/internal/core/agent"
	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
	domaintools "github.com/boxify/api-go/internal/domain/tools"
	"github.com/boxify/api-go/internal/domain/types"
	"github.com/boxify/api-go/internal/infrastructure/realtime"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

const defaultChatTemperature = 0.7

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
	if l.svcCtx == nil || l.svcCtx.Realtime == nil {
		return nil, xerr.Internal("实时消息服务未初始化", nil)
	}

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
	if err != nil {
		return nil, err
	}
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
	userMessage, err := l.svcCtx.MessageRepo.Create(l.ctx, userID, &models.Message{
		ConversationID: conversation.ID,
		Role:           string(llm.UserRole),
		Content:        userText,
		MetaData:       &models.MessageMetaData{ImageKeys: append([]string(nil), input.ImageKeys...)},
	})
	if err != nil {
		return nil, err
	}

	topic := realtime.ConversationTopic(conversation.ID)
	subscription, err := l.svcCtx.Realtime.Subscribe(l.ctx, topic)
	if err != nil {
		return nil, err
	}

	// 创建事件通道并启动后台 goroutine 转发事件
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

	// 启动后台 goroutine 生成回复
	go func() {
		// TODO: 后续补 Redis 回合锁和断线续传缓冲，避免同会话并发生成。
		l.runChatTurnBG(context.WithoutCancel(l.ctx), userID, conversation.ID, userMessage.ID, input, attachments)
	}()
	return events, nil
}

// runChatTurnBG 后台生成回复
func (l *ChatStreamLogic) runChatTurnBG(
	ctx context.Context,
	userID uuid.UUID,
	conversationID uuid.UUID,
	userMessageID uuid.UUID,
	input *request.ChatStreamRequest,
	attachments []*types.MessageAttachment,
) {
	topic := realtime.ConversationTopic(conversationID)
	if _, err := l.svcCtx.ConversationRepo.FindByID(ctx, userID, conversationID); err != nil {
		_ = l.svcCtx.Realtime.Publish(ctx, topic, types.NewErrorEvent("会话不存在"))
		return
	}

	result, err := l.generateAssistant(ctx, userID, conversationID, userMessageID, input, attachments)
	if err != nil {
		if strings.TrimSpace(result.Partial) != "" {
			if _, saveErr := l.svcCtx.MessageRepo.Create(ctx, userID, &models.Message{
				ConversationID: conversationID,
				Role:           string(llm.AssistantRole),
				Content:        strings.TrimSpace(result.Partial),
				MetaData:       &models.MessageMetaData{ToolCalls: result.ToolCalls, Interrupted: true},
			}); saveErr != nil {
				l.log.WarnContext(ctx, "保存部分回复失败", slog.String("error", saveErr.Error()))
			}
		}
		l.log.WarnContext(ctx, "后台生成回复失败", slog.String("error", err.Error()))
		_ = l.svcCtx.Realtime.Publish(ctx, topic, types.NewErrorEvent("生成失败："+err.Error()))
		return
	}

	answer := strings.TrimSpace(result.Answer)
	assistantMsg, err := l.svcCtx.MessageRepo.Create(ctx, userID, &models.Message{
		ConversationID: conversationID,
		Role:           string(llm.AssistantRole),
		Content:        answer,
		MetaData:       &models.MessageMetaData{ToolCalls: result.ToolCalls},
	})
	if err != nil {
		l.log.WarnContext(ctx, "保存AI回复失败", slog.String("error", err.Error()))
		_ = l.svcCtx.Realtime.Publish(ctx, topic, types.NewErrorEvent("保存回复失败："+err.Error()))
		return
	}
	// TODO: 后续在 ConversationRepository 增加 touch 能力，生成完成后刷新会话 updated_at。
	if answer != "" {
		_ = l.svcCtx.Realtime.Publish(ctx, topic, types.NewTokenEvent(answer))
	}
	_ = l.svcCtx.Realtime.Publish(ctx, topic, types.NewDoneEvent(assistantMsg.ID.String()))
	// TODO: 后续接入记忆萃取、图片入库、情绪分析等副作用，失败不能影响主回复。
}

type chatGenerationResult struct {
	Answer    string
	Partial   string
	ToolCalls []models.MessageToolCallMeta
}

type chatRuntimeConfig struct {
	EnableKnowledge bool
	Temperature     float64
}

// generateAssistant 生成AI回复
func (l *ChatStreamLogic) generateAssistant(
	ctx context.Context,
	userID uuid.UUID,
	conversationID uuid.UUID,
	userMessageID uuid.UUID,
	input *request.ChatStreamRequest,
	attachments []*types.MessageAttachment,
) (chatGenerationResult, error) {
	client, err := svc.ChatClient(ctx, l.svcCtx, userID)
	if err != nil {
		return chatGenerationResult{}, err
	}
	agentConfig, err := l.chatAgentConfig(ctx, userID)
	if err != nil {
		return chatGenerationResult{}, err
	}
	runtimeConfig := resolveChatRuntimeConfig(input, agentConfig)
	history, err := l.historyMessages(ctx, userID, conversationID, userMessageID)
	if err != nil {
		return chatGenerationResult{}, err
	}
	runCtx, kbIDs, err := chatToolContext(ctx, l.svcCtx, userID, runtimeConfig.EnableKnowledge)
	if err != nil {
		return chatGenerationResult{}, err
	}
	registry, err := l.chatToolRegistry(runCtx, kbIDs)
	if err != nil {
		return chatGenerationResult{}, err
	}

	hooks := &chatAgentHooks{}
	options := []coreagent.Option{
		coreagent.WithHooks(hooks),
		coreagent.WithModelOptions(llm.WithTemperature(runtimeConfig.Temperature)),
	}
	if agentConfig != nil && strings.TrimSpace(agentConfig.SystemPrompt) != "" {
		options = append(options, coreagent.WithSystemPrompt(strings.TrimSpace(agentConfig.SystemPrompt)))
	}
	// TODO: 接入 SkillID 后，将技能 prompt 叠加到 system prompt，并限制工具/知识库范围。
	// TODO: 接入 ImageKeys 后，按多模态模型路径生成回复；当前仅把 image_keys 保存到用户消息 metadata。
	// TODO: 接入 EnableMemory/EnableWebSearch 后，分别注入记忆召回和联网搜索工具。
	result, err := coreagent.New(client, registry, options...).Run(runCtx, coreagent.Input{
		Query:    composeChatQuery(input.Message, attachments),
		Messages: history,
	})
	out := chatGenerationResult{
		Partial:   hooks.lastModelOutput,
		ToolCalls: toolCallsFromAgentSteps(result),
	}
	if err != nil {
		if out.Partial == "" && result != nil {
			out.Partial = result.Answer
		}
		return out, err
	}
	if result != nil {
		out.Answer = result.Answer
	}
	return out, nil
}

// chatAgentConfig 获取Agent配置
func (l *ChatStreamLogic) chatAgentConfig(ctx context.Context, userID uuid.UUID) (*models.AgentConfig, error) {
	if l.svcCtx == nil || l.svcCtx.AgentConfigRepo == nil {
		return nil, nil
	}
	config, err := l.svcCtx.AgentConfigRepo.FindByUserID(ctx, userID)
	if err != nil {
		if xerr.From(err).Kind == xerr.KindNotFound {
			// TODO: 后续补默认 AgentConfig 初始化策略，避免每轮靠代码默认值兜底。
			return nil, nil
		}
		return nil, err
	}
	return config, nil
}

// historyMessages 获取会话历史消息
func (l *ChatStreamLogic) historyMessages(
	ctx context.Context,
	userID uuid.UUID,
	conversationID uuid.UUID,
	currentUserMessageID uuid.UUID,
) ([]*llm.Message, error) {
	rows, err := l.svcCtx.MessageRepo.ListByConversationID(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	messages := make([]*llm.Message, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.ID == currentUserMessageID || strings.TrimSpace(row.Content) == "" {
			continue
		}
		switch row.Role {
		case string(llm.UserRole):
			messages = append(messages, llm.UserMessage(row.Content))
		case string(llm.AssistantRole):
			messages = append(messages, llm.AssistantMessage(row.Content))
		}
	}
	return messages, nil
}

// chatToolRegistry 获取工具注册表
func (l *ChatStreamLogic) chatToolRegistry(
	ctx context.Context,
	kbIDs []uuid.UUID,
) (*coretool.Registry, error) {
	if l.svcCtx == nil {
		return coretool.NewRegistry(), nil
	}
	catalog, err := domaintools.NewCatalog(l.svcCtx)
	if err != nil {
		return nil, err
	}
	setNames := []string{domaintools.ToolSetSystem}
	if len(kbIDs) > 0 {
		setNames = append(setNames, domaintools.ToolSetKnowledge)
	}
	return catalog.BuildRegistry(ctx, coretool.Selection{SetNames: setNames})
}

// chatToolContext 获取工具上下文
func chatToolContext(
	ctx context.Context,
	svcCtx *svc.ServiceContext,
	userID uuid.UUID,
	enableKnowledge bool,
) (context.Context, []uuid.UUID, error) {
	ctx = util.WithUserID(ctx, userID)
	if !enableKnowledge || svcCtx == nil || svcCtx.KnowledgeBaseRepo == nil {
		return ctx, nil, nil
	}
	rows, err := svcCtx.KnowledgeBaseRepo.List(ctx, userID)
	if err != nil {
		return ctx, nil, err
	}
	kbIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.ChatEnabled && row.ID != uuid.Nil {
			kbIDs = append(kbIDs, row.ID)
		}
	}
	if len(kbIDs) == 0 {
		return ctx, nil, nil
	}
	return util.WithKnowledgeBaseIDs(ctx, kbIDs), kbIDs, nil
}

// resolveChatRuntimeConfig 归一化聊天运行参数，并在逻辑层提供业务默认值兜底。
func resolveChatRuntimeConfig(input *request.ChatStreamRequest, agentConfig *models.AgentConfig) chatRuntimeConfig {
	config := chatRuntimeConfig{
		EnableKnowledge: false,
		Temperature:     defaultChatTemperature,
	}
	if agentConfig != nil && agentConfig.Temperature > 0 {
		config.Temperature = agentConfig.Temperature
	}
	if agentConfig != nil {
		config.EnableKnowledge = agentConfig.EnableKnowledge
	}
	if input != nil && input.EnableKnowledge != nil {
		// 请求层显式值优先级最高；false 必须能覆盖 AgentConfig 默认开启。
		config.EnableKnowledge = *input.EnableKnowledge
	}
	return config
}

// composeChatQuery 组合聊天查询
func composeChatQuery(message string, attachments []*types.MessageAttachment) string {
	query := strings.TrimSpace(message)
	if len(attachments) == 0 {
		return query
	}
	parts := []string{query, "\n\n附件内容："}
	for _, attachment := range attachments {
		if attachment == nil || strings.TrimSpace(attachment.Content) == "" {
			continue
		}
		name := strings.TrimSpace(attachment.FileName)
		if name == "" {
			name = "未命名附件"
		}
		parts = append(parts, "\n["+name+"]\n"+strings.TrimSpace(attachment.Content))
	}
	return strings.TrimSpace(strings.Join(parts, ""))
}

// toolCallsFromAgentSteps 从Agent步骤中提取工具调用
func toolCallsFromAgentSteps(result *coreagent.Result) []models.MessageToolCallMeta {
	if result == nil {
		return nil
	}
	out := make([]models.MessageToolCallMeta, 0, len(result.Steps))
	for _, step := range result.Steps {
		if strings.TrimSpace(step.Action) == "" {
			continue
		}
		out = append(out, models.MessageToolCallMeta{
			Tool:        step.Action,
			Input:       map[string]any(step.ActionInput),
			Observation: step.Observation,
			Iteration:   step.Iteration,
		})
	}
	return out
}

type chatAgentHooks struct {
	coreagent.NoopHooks
	lastModelOutput string
}

// AfterModel 调用模型后钩子
func (h *chatAgentHooks) AfterModel(
	ctx context.Context,
	state coreagent.State,
	output string,
	modelErr error,
) error {
	if strings.TrimSpace(output) != "" {
		h.lastModelOutput = strings.TrimSpace(output)
	}
	return nil
}

// ensureConversation 确保会话存在
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
