package llm

import coretool "github.com/boxify/api-go/internal/core/tool"

// UserMessage 创建用户角色消息。
func UserMessage(content string) *Message {
	return &Message{
		Role:    UserRole,
		Content: content,
	}
}

// AssistantMessage 创建助手角色消息。
func AssistantMessage(content string) *Message {
	return &Message{
		Role:    AssistantRole,
		Content: content,
	}
}

// SystemMessage 创建系统角色消息。
func SystemMessage(content string) *Message {
	return &Message{
		Role:    SystemRole,
		Content: content,
	}
}

// CloneMessages 返回消息列表的独立副本。
//
// nil 输入返回 nil；列表中的 nil 消息会原样保留。ToolCalls 会被深拷贝，避免调用方
// 后续修改工具参数污染已构造的模型请求。
func CloneMessages(messages []*Message) []*Message {
	if messages == nil {
		return nil
	}
	out := make([]*Message, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			out = append(out, nil)
			continue
		}
		copied := *message
		copied.ToolCalls = cloneToolCalls(message.ToolCalls)
		out = append(out, &copied)
	}
	return out
}

func cloneToolCalls(calls []LLMToolCall) []LLMToolCall {
	if calls == nil {
		return nil
	}
	out := make([]LLMToolCall, len(calls))
	copy(out, calls)
	for i := range out {
		out[i].Input = cloneToolInput(out[i].Input)
	}
	return out
}

func cloneToolInput(input coretool.Input) coretool.Input {
	if input == nil {
		return nil
	}
	out := make(coretool.Input, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
