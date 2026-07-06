package agent

import (
	"context"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

// Hooks 定义 Agent 关键生命周期和状态迁移钩子。
type Hooks interface {
	BeforeRun(ctx context.Context, state State) error
	AfterRun(ctx context.Context, result Result, runErr error) error
	BeforeTransition(ctx context.Context, state State, transition Transition) error
	AfterTransition(ctx context.Context, state State, transition Transition) error
	BeforeModel(ctx context.Context, state State, messages []*llm.Message) error
	AfterModel(ctx context.Context, state State, output string, modelErr error) error
	AfterParse(ctx context.Context, state State, decision Decision, parseErr error) error
	BeforeTool(ctx context.Context, state State, call ToolCall) error
	AfterTool(ctx context.Context, state State, call ToolCall, output coretool.Output, toolErr error) error
	OnStep(ctx context.Context, state State, step Step) error
	OnError(ctx context.Context, state State, err error) error
}

// NoopHooks 是不执行任何副作用的默认 hooks。
type NoopHooks struct{}

// BeforeRun 在运行开始前调用。
func (NoopHooks) BeforeRun(ctx context.Context, state State) error { return nil }

// AfterRun 在运行结束后调用。
func (NoopHooks) AfterRun(ctx context.Context, result Result, runErr error) error { return nil }

// BeforeTransition 在状态迁移前调用。
func (NoopHooks) BeforeTransition(ctx context.Context, state State, transition Transition) error {
	return nil
}

// AfterTransition 在状态迁移后调用。
func (NoopHooks) AfterTransition(ctx context.Context, state State, transition Transition) error {
	return nil
}

// BeforeModel 在模型调用前调用。
func (NoopHooks) BeforeModel(ctx context.Context, state State, messages []*llm.Message) error {
	return nil
}

// AfterModel 在模型调用后调用。
func (NoopHooks) AfterModel(ctx context.Context, state State, output string, modelErr error) error {
	return nil
}

// AfterParse 在模型输出解析后调用。
func (NoopHooks) AfterParse(ctx context.Context, state State, decision Decision, parseErr error) error {
	return nil
}

// BeforeTool 在工具调用前调用。
func (NoopHooks) BeforeTool(ctx context.Context, state State, call ToolCall) error { return nil }

// AfterTool 在工具调用后调用。
func (NoopHooks) AfterTool(ctx context.Context, state State, call ToolCall, output coretool.Output, toolErr error) error {
	return nil
}

// OnStep 在单次 ReAct step 记录后调用。
func (NoopHooks) OnStep(ctx context.Context, state State, step Step) error { return nil }

// OnError 在运行错误发生后调用。
func (NoopHooks) OnError(ctx context.Context, state State, err error) error { return nil }
