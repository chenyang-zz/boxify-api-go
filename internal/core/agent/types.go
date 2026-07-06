package agent

import (
	"context"
	"errors"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

// ErrMaxIterations 表示 Agent 在达到最大迭代次数后仍未得到最终答案。
var ErrMaxIterations = errors.New("max iterations reached")

// ErrParseDecision 表示模型输出不符合 ReAct 决策格式。
var ErrParseDecision = errors.New("parse react decision")

// ErrInvalidActionInput 表示 Action Input 不是合法 JSON object。
var ErrInvalidActionInput = errors.New("invalid action input")

// ErrToolCallingUnsupported 表示模型客户端当前不支持原生工具调用。
var ErrToolCallingUnsupported = errors.New("tool calling unsupported")

// Input 表示一次 Agent 运行的输入。
//
// Query 用于最常见的单轮用户问题。Messages 用于调用方传入已有上下文；
// 两者同时存在时，默认 prompt builder 会同时保留 Messages 并追加 Query。
type Input struct {
	Query    string
	Messages []*llm.Message
}

// Result 表示一次 Agent 运行的结果。
type Result struct {
	Answer     string
	Steps      []Step
	Iterations int
	StoppedBy  StopReason
}

// StopReason 表示 Agent 停止运行的原因。
type StopReason string

const (
	// StopFinalAnswer 表示 Agent 得到了最终答案。
	StopFinalAnswer StopReason = "final_answer"
	// StopMaxIterations 表示 Agent 达到最大迭代次数。
	StopMaxIterations StopReason = "max_iterations"
	// StopError 表示 Agent 因错误停止。
	StopError StopReason = "error"
)

// Phase 表示 Agent 内部状态机阶段。
type Phase string

const (
	// PhaseStart 表示运行刚初始化。
	PhaseStart Phase = "start"
	// PhaseBuildPrompt 表示正在构造模型输入。
	PhaseBuildPrompt Phase = "build_prompt"
	// PhaseModel 表示正在调用模型。
	PhaseModel Phase = "model"
	// PhaseFallback 表示 function calling 路径退回文本 ReAct。
	PhaseFallback Phase = "fallback"
	// PhaseParse 表示正在解析模型输出。
	PhaseParse Phase = "parse"
	// PhaseTool 表示正在调用工具。
	PhaseTool Phase = "tool"
	// PhaseObserve 表示正在记录工具观察结果。
	PhaseObserve Phase = "observe"
	// PhaseFinish 表示运行正常结束。
	PhaseFinish Phase = "finish"
	// PhaseError 表示运行因错误结束。
	PhaseError Phase = "error"
)

// Transition 表示一次状态阶段迁移。
type Transition struct {
	From      Phase
	To        Phase
	Iteration int
	Reason    string
}

// State 表示 Agent 当前运行状态。
//
// State 会传给 prompt builder 和 hooks。调用方应把它视为快照，不应依赖修改它来影响
// Agent 内部状态。
type State struct {
	Input        Input
	Tools        []coretool.Descriptor
	Steps        []Step
	Iteration    int
	Phase        Phase
	LastDecision Decision
	LastError    error
	SystemPrompt string
}

// Step 表示一次 Agent 迭代中的模型决策、工具调用和观察结果。
type Step struct {
	Iteration      int
	Thought        string
	Action         string
	ActionInput    coretool.Input
	ToolCallID     string
	Observation    string
	FinalAnswer    string
	RawModelOutput string
}

// DecisionKind 表示模型输出被解析后的决策类型。
type DecisionKind string

const (
	// DecisionFinal 表示模型给出了最终答案。
	DecisionFinal DecisionKind = "final"
	// DecisionToolCall 表示模型要求调用工具。
	DecisionToolCall DecisionKind = "tool_call"
)

// Decision 表示模型输出被标准化后的结构化决策。
type Decision struct {
	Kind        DecisionKind
	Thought     string
	FinalAnswer string
	Action      string
	ActionInput coretool.Input
	ToolCallID  string
}

// ToolCall 表示 Agent 准备执行的一次工具调用。
type ToolCall struct {
	Name  string
	Input coretool.Input
}

// Planner 负责把当前状态转成下一步标准化决策。
type Planner interface {
	Plan(ctx context.Context, state State, opts ...llm.ModelCallOption) (Decision, error)
}

// ToolCallingPlanner 表示支持模型原生工具调用的 planner。
type ToolCallingPlanner interface {
	Planner
	SupportsToolCalling() bool
}

// ToolCallingClient 是 agent 消费侧定义的原生工具调用模型接口。
//
// 具体供应商 adapter 负责把 ToolCallingInput 转换成 OpenAI、Anthropic 或其他模型
// 的消息格式，并把模型返回的 tool call 转成 ToolCallingOutput。
type ToolCallingClient interface {
	InvokeWithTools(ctx context.Context, input ToolCallingInput, opts ...llm.ModelCallOption) (ToolCallingOutput, error)
}

// ToolCallingInput 表示一次原生工具调用模型请求。
type ToolCallingInput struct {
	Messages []*llm.Message
	Tools    []coretool.Descriptor
	Steps    []Step
}

// ToolCallingOutput 表示原生工具调用模型输出。
type ToolCallingOutput struct {
	Content   string
	ToolCalls []ModelToolCall
}

// ModelToolCall 表示模型请求执行的一次工具调用。
type ModelToolCall struct {
	ID    string
	Name  string
	Input coretool.Input
}
