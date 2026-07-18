package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

// CloneDecisionFunc 复制具体 Agent 实现的决策对象。
type CloneDecisionFunc[D any] func(D) D

// CloneStepFunc 复制具体 Agent 实现的步骤对象。
type CloneStepFunc[S any] func(S) S

// Base 承载 Agent 通用运行骨架。
//
// Base 不理解具体决策协议，只负责依赖持有、状态快照、生命周期 hooks、工具执行和错误收尾。
type Base[D any, S any] struct {
	client              llm.Client
	registry            *coretool.Registry
	toolRunner          *coretool.Runner
	hooks               Hooks[D, S]
	maxIterations       int
	observationMaxRunes int
	systemPrompt        string
	modelOptions        []llm.ModelCallOption
	cloneDecision       CloneDecisionFunc[D]
	cloneStep           CloneStepFunc[S]
}

// NewBase 创建带默认值的 Base。
//
// registry 为 nil 时会创建空工具注册表。cloneDecision 和 cloneStep 为 nil 时使用值拷贝。
func NewBase[D any, S any](client llm.Client, registry *coretool.Registry, opts ...Option[D, S]) *Base[D, S] {
	if registry == nil {
		registry = coretool.NewRegistry()
	}
	base := &Base[D, S]{
		client:              client,
		registry:            registry,
		toolRunner:          coretool.NewRunner(registry),
		hooks:               NoopHooks[D, S]{},
		maxIterations:       DefaultMaxIterations,
		observationMaxRunes: DefaultObservationMaxRunes,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(base)
		}
	}
	return base
}

// SetCloneFuncs 设置具体决策和步骤的深拷贝函数。
func (b *Base[D, S]) SetCloneFuncs(decision CloneDecisionFunc[D], step CloneStepFunc[S]) {
	if decision != nil {
		b.cloneDecision = decision
	}
	if step != nil {
		b.cloneStep = step
	}
}

// Client 返回模型客户端。
func (b *Base[D, S]) Client() llm.Client {
	if b == nil {
		return nil
	}
	return b.client
}

// Hooks 返回生命周期 hooks。
func (b *Base[D, S]) Hooks() Hooks[D, S] {
	if b == nil || b.hooks == nil {
		return NoopHooks[D, S]{}
	}
	return b.hooks
}

// NewState 创建一次运行的初始状态。
func (b *Base[D, S]) NewState(input Input) State[D, S] {
	state := State[D, S]{
		Input:        input,
		Phase:        PhaseStart,
		SystemPrompt: b.systemPrompt,
	}
	if b != nil && b.registry != nil {
		state.Tools = b.registry.List(nil)
	}
	return state
}

// Validate reports whether Base 具备运行所需依赖。
func (b *Base[D, S]) Validate() error {
	if b == nil {
		return errors.New("agent base is nil")
	}
	if b.client == nil {
		return errors.New("agent model client is nil")
	}
	return nil
}

// RunConfig 合并默认运行配置和单次运行配置。
func (b *Base[D, S]) RunConfig(opts ...RunOption) RunConfig {
	cfg := RunConfig{
		MaxIterations: b.maxIterations,
		ModelOptions:  append([]llm.ModelCallOption{}, b.modelOptions...),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

// Transition 执行状态迁移并触发迁移 hooks。
func (b *Base[D, S]) Transition(ctx context.Context, state *State[D, S], to Phase, reason string) error {
	transition := Transition{
		From:      state.Phase,
		To:        to,
		Iteration: state.Iteration,
		Reason:    reason,
	}
	if err := b.Hooks().BeforeTransition(ctx, b.CloneState(*state), transition); err != nil {
		return err
	}
	state.Phase = to
	if err := b.Hooks().AfterTransition(ctx, b.CloneState(*state), transition); err != nil {
		return err
	}
	return nil
}

// InvokeTool 调用工具并返回工具输出。
func (b *Base[D, S]) InvokeTool(ctx context.Context, name string, input coretool.Input) (coretool.Output, error) {
	if b == nil || b.toolRunner == nil {
		return coretool.Output{}, errors.New("agent tool runner is nil")
	}
	return b.toolRunner.Invoke(ctx, name, input)
}

// TruncateObservation 按 Base 配置裁剪工具观察结果。
func (b *Base[D, S]) TruncateObservation(value string) string {
	if b == nil {
		return value
	}
	return truncateRunes(value, b.observationMaxRunes)
}

// FinishWithError 处理错误并完成运行。
func (b *Base[D, S]) FinishWithError(ctx context.Context, state State[D, S], result Result[S], err error) (*Result[S], error) {
	state.LastError = err
	result.StoppedBy = stopReasonForError(err)
	if transitionErr := b.Transition(ctx, &state, PhaseError, "error"); transitionErr != nil {
		err = fmt.Errorf("%w: %v", err, transitionErr)
	}
	_ = b.Hooks().OnError(ctx, b.CloneState(state), err)
	_ = b.Hooks().AfterRun(ctx, result, err)
	return &result, err
}

// CloneState 返回状态快照，避免 hooks 或 prompt builder 修改内部状态。
func (b *Base[D, S]) CloneState(state State[D, S]) State[D, S] {
	state.Input.Messages = llm.CloneMessages(state.Input.Messages)
	state.Tools = coretool.CloneDescriptors(state.Tools)
	state.Steps = b.CloneSteps(state.Steps)
	if b != nil && b.cloneDecision != nil {
		state.LastDecision = b.cloneDecision(state.LastDecision)
	}
	return state
}

// CloneSteps 返回步骤快照。
func (b *Base[D, S]) CloneSteps(steps []S) []S {
	if steps == nil {
		return nil
	}
	out := make([]S, len(steps))
	for i, step := range steps {
		if b != nil && b.cloneStep != nil {
			out[i] = b.cloneStep(step)
			continue
		}
		out[i] = step
	}
	return out
}

// CloneResult 返回运行结果快照。
func (b *Base[D, S]) CloneResult(result Result[S]) Result[S] {
	result.Steps = b.CloneSteps(result.Steps)
	return result
}

func stopReasonForError(err error) StopReason {
	if errors.Is(err, ErrMaxIterations) {
		return StopMaxIterations
	}
	return StopError
}
