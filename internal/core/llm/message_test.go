package llm

import "testing"

// 验证点：CloneMessages 应复制消息切片和元素，且保留 nil 消息位置。
func TestCloneMessagesReturnsIndependentCopy(t *testing.T) {
	messages := []*Message{
		UserMessage("hello"),
		nil,
		AssistantMessage("world"),
	}

	cloned := CloneMessages(messages)
	messages[0].Content = "changed"
	messages[2] = SystemMessage("system")

	if len(cloned) != 3 {
		t.Fatalf("CloneMessages() len = %d, want 3", len(cloned))
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
}
