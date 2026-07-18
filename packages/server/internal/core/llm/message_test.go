package llm

import "testing"

// 验证点：CloneMessages 应复制消息切片和元素，且保留 nil 消息位置。
func TestCloneMessagesReturnsIndependentCopy(t *testing.T) {
	messages := []*Message{
		UserMessage("hello"),
		nil,
		{
			Role:    AssistantRole,
			Content: "world",
			ToolCalls: []LLMToolCall{
				{ID: "call_1", Name: "search", Input: map[string]any{"query": "golang"}, RawInput: `{"query":"golang"}`},
			},
		},
		{
			Role:       ToolRole,
			Content:    "result",
			ToolCallID: "call_1",
			ToolName:   "search",
		},
	}

	cloned := CloneMessages(messages)
	messages[0].Content = "changed"
	messages[2].ToolCalls[0].Input["query"] = "changed"
	messages[2].ToolCalls[0].Name = "changed"
	messages[3].ToolCallID = "changed"

	if len(cloned) != 4 {
		t.Fatalf("CloneMessages() len = %d, want 4", len(cloned))
	}
	if cloned[0].Content != "hello" || cloned[0].Role != UserRole {
		t.Fatalf("CloneMessages()[0] = %#v, want original user hello", cloned[0])
	}
	if cloned[1] != nil {
		t.Fatalf("CloneMessages()[1] = %#v, want nil", cloned[1])
	}
	if cloned[2].Content != "world" || cloned[2].Role != AssistantRole {
		t.Fatalf("CloneMessages()[2] = %#v, want original assistant world", cloned[2])
	}
	if len(cloned[2].ToolCalls) != 1 || cloned[2].ToolCalls[0].Name != "search" || cloned[2].ToolCalls[0].Input["query"] != "golang" {
		t.Fatalf("CloneMessages()[2].ToolCalls = %#v, want independent search call", cloned[2].ToolCalls)
	}
	if cloned[3].Role != ToolRole || cloned[3].ToolCallID != "call_1" || cloned[3].ToolName != "search" {
		t.Fatalf("CloneMessages()[3] = %#v, want original tool result metadata", cloned[3])
	}
}
