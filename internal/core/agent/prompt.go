package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

const defaultSystemPrompt = `You are a ReAct agent. Use tools when needed.

When you need a tool, respond exactly with:
Thought: your reasoning
Action: tool_name
Action Input: {"key":"value"}

When you can answer, respond exactly with:
Thought: your reasoning
Final Answer: your final answer`

// PromptBuilder 构造每轮模型输入消息。
type PromptBuilder interface {
	Build(ctx context.Context, state State) ([]*llm.Message, error)
}

// ReActPromptBuilder 是默认 ReAct prompt builder。
type ReActPromptBuilder struct{}

// NewReActPromptBuilder 创建默认 ReAct prompt builder。
func NewReActPromptBuilder() *ReActPromptBuilder {
	return &ReActPromptBuilder{}
}

// Build 根据当前状态构造模型消息。
func (b *ReActPromptBuilder) Build(ctx context.Context, state State) ([]*llm.Message, error) {
	system := strings.TrimSpace(state.SystemPrompt)
	if system == "" {
		system = defaultSystemPrompt
	}
	system = system + "\n\nAvailable tools:\n" + formatTools(state.Tools)

	messages := []*llm.Message{llm.SystemMessage(system)}
	messages = append(messages, cloneLLMMessages(state.Input.Messages)...)
	userContent := strings.TrimSpace(state.Input.Query)
	if userContent != "" {
		messages = append(messages, llm.UserMessage(userContent))
	}
	if len(state.Steps) > 0 {
		messages = append(messages, llm.AssistantMessage(formatScratchpad(state.Steps)))
	}
	return messages, nil
}

func formatTools(tools []coretool.Descriptor) string {
	if len(tools) == 0 {
		return "(none)"
	}
	lines := make([]string, 0, len(tools))
	for _, item := range tools {
		schema, _ := json.Marshal(item.Schema.Parameters)
		lines = append(lines, fmt.Sprintf("- %s: %s\n  parameters: %s", item.Name, item.Description, string(schema)))
	}
	return strings.Join(lines, "\n")
}

func formatScratchpad(steps []Step) string {
	parts := make([]string, 0, len(steps))
	for _, step := range steps {
		if step.FinalAnswer != "" {
			parts = append(parts, fmt.Sprintf("Thought: %s\nFinal Answer: %s", step.Thought, step.FinalAnswer))
			continue
		}
		input, _ := json.Marshal(step.ActionInput)
		parts = append(parts, fmt.Sprintf(
			"Thought: %s\nAction: %s\nAction Input: %s\nObservation: %s",
			step.Thought,
			step.Action,
			string(input),
			step.Observation,
		))
	}
	return strings.Join(parts, "\n\n")
}
