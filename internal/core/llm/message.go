package llm

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
// nil 输入返回 nil；列表中的 nil 消息会原样保留。Message 当前只包含值类型字段，
// 因此逐项复制结构体即可隔离调用方后续修改。
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
		out = append(out, &copied)
	}
	return out
}
