package react

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/boxify/api-go/internal/core/llm"
)

type plannerResult struct {
	Decision Decision
	Output   string
	Fallback bool
}

type tracePlanner interface {
	planTrace(ctx context.Context, state State, opts ...llm.ModelCallOption) (plannerResult, error)
}

type modelMessagePlanner interface {
	modelMessages(ctx context.Context, state State) ([]*llm.Message, error)
}

// ReActTextPlanner 使用文本 ReAct prompt 和 parser 生成决策。
type ReActTextPlanner struct {
	client        llm.Client
	promptBuilder PromptBuilder
	parser        Parser
}

// NewReActTextPlanner 创建文本 ReAct planner。
//
// builder 或 parser 为 nil 时会使用默认实现。client 为 nil 时，Plan 会返回错误。
func NewReActTextPlanner(client llm.Client, builder PromptBuilder, parser Parser) *ReActTextPlanner {
	if builder == nil {
		builder = NewReActPromptBuilder()
	}
	if parser == nil {
		parser = NewReActParser()
	}
	return &ReActTextPlanner{
		client:        client,
		promptBuilder: builder,
		parser:        parser,
	}
}

// Plan 调用文本模型并解析 ReAct 决策。
func (p *ReActTextPlanner) Plan(ctx context.Context, state State, opts ...llm.ModelCallOption) (Decision, error) {
	result, err := p.planTrace(ctx, state, opts...)
	if err != nil {
		return Decision{}, err
	}
	return result.Decision, nil
}

func (p *ReActTextPlanner) planTrace(ctx context.Context, state State, opts ...llm.ModelCallOption) (plannerResult, error) {
	if p == nil {
		return plannerResult{}, errors.New("react text planner is nil")
	}
	if p.client == nil {
		return plannerResult{}, errors.New("agent model client is nil")
	}
	messages, err := p.promptBuilder.Build(ctx, cloneState(state))
	if err != nil {
		return plannerResult{}, err
	}
	output, err := p.client.Invoke(ctx, messages, opts...)
	if err != nil {
		return plannerResult{Output: output}, err
	}
	decision, err := p.parser.Parse(ctx, output)
	if err != nil {
		return plannerResult{Output: output}, err
	}
	return plannerResult{
		Decision: decision,
		Output:   output,
	}, nil
}

func (p *ReActTextPlanner) modelMessages(ctx context.Context, state State) ([]*llm.Message, error) {
	if p == nil {
		return nil, errors.New("react text planner is nil")
	}
	return p.promptBuilder.Build(ctx, cloneState(state))
}

// FunctionCallingPlanner 使用模型原生工具调用能力生成决策。
type FunctionCallingPlanner struct {
	client ToolCallingClient
}

// NewFunctionCallingPlanner 创建 function calling planner。
func NewFunctionCallingPlanner(client ToolCallingClient) *FunctionCallingPlanner {
	return &FunctionCallingPlanner{client: client}
}

// SupportsToolCalling reports whether planner 持有可用的 ToolCallingClient。
func (p *FunctionCallingPlanner) SupportsToolCalling() bool {
	return p != nil && p.client != nil
}

// Plan 调用支持工具调用的模型，并把输出规整为统一 Decision。
func (p *FunctionCallingPlanner) Plan(ctx context.Context, state State, opts ...llm.ModelCallOption) (Decision, error) {
	result, err := p.planTrace(ctx, state, opts...)
	if err != nil {
		return Decision{}, err
	}
	return result.Decision, nil
}

func (p *FunctionCallingPlanner) planTrace(ctx context.Context, state State, opts ...llm.ModelCallOption) (plannerResult, error) {
	if !p.SupportsToolCalling() {
		return plannerResult{}, ErrToolCallingUnsupported
	}
	input := ToolCallingInput{
		Messages: toolCallingMessages(state.Input),
		Tools:    cloneToolDescriptors(state.Tools),
		Steps:    cloneSteps(state.Steps),
	}
	output, err := p.client.InvokeWithTools(ctx, input, opts...)
	if err != nil {
		return plannerResult{Output: outputSummary(output)}, err
	}
	decision, err := decisionFromToolCallingOutput(output)
	if err != nil {
		return plannerResult{Output: outputSummary(output)}, err
	}
	return plannerResult{
		Decision: decision,
		Output:   outputSummary(output),
	}, nil
}

func (p *FunctionCallingPlanner) modelMessages(ctx context.Context, state State) ([]*llm.Message, error) {
	return toolCallingMessages(state.Input), nil
}

// AutoPlanner 按配置在 function calling 和文本 ReAct 之间选择执行路径。
type AutoPlanner struct {
	toolPlanner        *FunctionCallingPlanner
	reactPlanner       *ReActTextPlanner
	toolCallingEnabled bool
	fallbackToReAct    bool
}

// NewAutoPlanner 创建自动选择 planner。
//
// 当 client 实现 ToolCallingClient 且 enabled 为 true 时优先使用 function calling；
// 否则直接使用文本 ReAct。function calling 返回 ErrToolCallingUnsupported 时，fallback
// 为 true 会自动退回文本 ReAct。
func NewAutoPlanner(client llm.Client, builder PromptBuilder, parser Parser, enabled bool, fallback bool) *AutoPlanner {
	var toolPlanner *FunctionCallingPlanner
	if toolClient, ok := client.(ToolCallingClient); ok {
		toolPlanner = NewFunctionCallingPlanner(toolClient)
	}
	return &AutoPlanner{
		toolPlanner:        toolPlanner,
		reactPlanner:       NewReActTextPlanner(client, builder, parser),
		toolCallingEnabled: enabled,
		fallbackToReAct:    fallback,
	}
}

// Plan 根据当前模型能力和配置生成决策。
func (p *AutoPlanner) Plan(ctx context.Context, state State, opts ...llm.ModelCallOption) (Decision, error) {
	result, err := p.planTrace(ctx, state, opts...)
	if err != nil {
		return Decision{}, err
	}
	return result.Decision, nil
}

func (p *AutoPlanner) planTrace(ctx context.Context, state State, opts ...llm.ModelCallOption) (plannerResult, error) {
	if p == nil {
		return plannerResult{}, errors.New("auto planner is nil")
	}
	if p.toolCallingEnabled && p.toolPlanner != nil && p.toolPlanner.SupportsToolCalling() {
		result, err := p.toolPlanner.planTrace(ctx, state, opts...)
		if err == nil {
			return result, nil
		}
		// 只有明确“不支持工具调用”才自动退回文本 ReAct，避免吞掉供应商调用错误。
		if !errors.Is(err, ErrToolCallingUnsupported) || !p.fallbackToReAct {
			return result, err
		}
		fallbackResult, fallbackErr := p.reactPlanner.planTrace(ctx, state, opts...)
		fallbackResult.Fallback = true
		return fallbackResult, fallbackErr
	}
	return p.reactPlanner.planTrace(ctx, state, opts...)
}

func (p *AutoPlanner) modelMessages(ctx context.Context, state State) ([]*llm.Message, error) {
	if p != nil && p.toolCallingEnabled && p.toolPlanner != nil && p.toolPlanner.SupportsToolCalling() {
		return p.toolPlanner.modelMessages(ctx, state)
	}
	return p.reactPlanner.modelMessages(ctx, state)
}

func toolCallingMessages(input Input) []*llm.Message {
	messages := llm.CloneMessages(input.Messages)
	if strings.TrimSpace(input.Query) == "" {
		return messages
	}
	return append(messages, &llm.Message{
		Role:    llm.UserRole,
		Content: input.Query,
	})
}

func decisionFromToolCallingOutput(output ToolCallingOutput) (Decision, error) {
	if len(output.ToolCalls) > 0 {
		call := output.ToolCalls[0]
		return Decision{
			Kind:        DecisionToolCall,
			Action:      strings.TrimSpace(call.Name),
			ActionInput: cloneInput(call.Input),
			ToolCallID:  call.ID,
		}, nil
	}
	if strings.TrimSpace(output.Content) == "" {
		return Decision{}, fmt.Errorf("%w: empty tool calling output", ErrParseDecision)
	}
	return Decision{
		Kind:        DecisionFinal,
		FinalAnswer: strings.TrimSpace(output.Content),
	}, nil
}

func outputSummary(output ToolCallingOutput) string {
	if len(output.ToolCalls) == 0 {
		return output.Content
	}
	names := make([]string, 0, len(output.ToolCalls))
	for _, call := range output.ToolCalls {
		names = append(names, call.Name)
	}
	return "tool_call:" + strings.Join(names, ",")
}

var _ Planner = (*ReActTextPlanner)(nil)
var _ Planner = (*FunctionCallingPlanner)(nil)
var _ ToolCallingPlanner = (*FunctionCallingPlanner)(nil)
var _ Planner = (*AutoPlanner)(nil)
var _ tracePlanner = (*ReActTextPlanner)(nil)
var _ tracePlanner = (*FunctionCallingPlanner)(nil)
var _ tracePlanner = (*AutoPlanner)(nil)
var _ modelMessagePlanner = (*ReActTextPlanner)(nil)
var _ modelMessagePlanner = (*FunctionCallingPlanner)(nil)
var _ modelMessagePlanner = (*AutoPlanner)(nil)
