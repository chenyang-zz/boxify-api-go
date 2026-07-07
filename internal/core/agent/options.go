package agent

import (
	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

const (
	// DefaultMaxIterations 是 Agent 默认最大迭代次数。
	DefaultMaxIterations = 6
	// DefaultObservationMaxRunes 是 Observation 默认最大 rune 数。
	DefaultObservationMaxRunes = 4000
)

// Option 配置 Base 的长期行为。
type Option[D any, S any] func(*Base[D, S])

// RunOption 配置单次运行行为。
type RunOption func(*RunConfig)

// RunConfig 表示单次运行合并后的通用配置。
type RunConfig struct {
	MaxIterations int
	ModelOptions  []llm.ModelCallOption
}

// WithMaxIterations 设置默认最大迭代次数，非正数会被忽略。
func WithMaxIterations[D any, S any](n int) Option[D, S] {
	return func(base *Base[D, S]) {
		if n > 0 {
			base.maxIterations = n
		}
	}
}

// WithSystemPrompt 设置默认系统提示词。
func WithSystemPrompt[D any, S any](prompt string) Option[D, S] {
	return func(base *Base[D, S]) {
		if prompt != "" {
			base.systemPrompt = prompt
		}
	}
}

// WithHooks 设置 Agent 生命周期 hooks，nil 会被忽略。
func WithHooks[D any, S any](hooks Hooks[D, S]) Option[D, S] {
	return func(base *Base[D, S]) {
		if hooks != nil {
			base.hooks = hooks
		}
	}
}

// WithModelOptions 设置默认模型调用参数。
func WithModelOptions[D any, S any](opts ...llm.ModelCallOption) Option[D, S] {
	return func(base *Base[D, S]) {
		base.modelOptions = append(base.modelOptions, opts...)
	}
}

// WithObservationMaxRunes 设置 Observation 最大 rune 数，非正数会被忽略。
func WithObservationMaxRunes[D any, S any](n int) Option[D, S] {
	return func(base *Base[D, S]) {
		if n > 0 {
			base.observationMaxRunes = n
		}
	}
}

// WithToolRunner 设置工具调用器，nil 会被忽略。
func WithToolRunner[D any, S any](runner *coretool.Runner) Option[D, S] {
	return func(base *Base[D, S]) {
		if runner != nil {
			base.toolRunner = runner
		}
	}
}

// WithRunMaxIterations 设置单次运行的最大迭代次数，非正数会被忽略。
func WithRunMaxIterations(n int) RunOption {
	return func(cfg *RunConfig) {
		if n > 0 {
			cfg.MaxIterations = n
		}
	}
}

// WithRunModelOptions 设置单次运行的模型调用参数。
func WithRunModelOptions(opts ...llm.ModelCallOption) RunOption {
	return func(cfg *RunConfig) {
		cfg.ModelOptions = append(cfg.ModelOptions, opts...)
	}
}
