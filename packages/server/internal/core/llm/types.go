package llm

// MessageRoleType 表示模型消息在对话中的角色。
type MessageRoleType string

const (
	// SystemRole 表示系统级指令消息。
	SystemRole MessageRoleType = "system"
	// UserRole 表示用户输入消息。
	UserRole MessageRoleType = "user"
	// AssistantRole 表示模型助手消息。
	AssistantRole MessageRoleType = "assistant"
	// ToolRole 表示工具调用结果消息。
	ToolRole MessageRoleType = "tool"
)

// Message 表示跨供应商归一化后的模型消息。
//
// Content 保存普通文本消息。ToolCalls 只在 assistant 消息中表示模型请求的工具调用。
// ToolCallID 和 ToolName 只在 tool 消息中表示该消息对应的工具调用结果。
type Message struct {
	Role       MessageRoleType `json:"role"`
	Content    string          `json:"content"`
	ToolCalls  []LLMToolCall   `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolName   string          `json:"tool_name,omitempty"`
}
