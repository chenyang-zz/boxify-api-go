# core/agent Agent 运行骨架与 ReAct 主循环

## Summary

`core/agent` 提供业务无关的 Agent 运行骨架，`core/agent/react` 在其上实现完整的 ReAct（Reasoning + Acting）主循环。

设计目标：

- **业务无关**：core 包只承载通用状态、生命周期 hooks、工具执行和错误收尾能力；具体的模型决策协议、prompt 构造和解析由子包实现，业务身份由调用方通过 `PromptBuilder` 注入。
- **双路径决策**：优先使用模型原生 function/tool calling 能力生成工具调用决策；当模型不支持或调用方显式关闭时，退回到文本 ReAct prompt 与 parser。两条路径最终被规整为统一的 `Decision` 和 `Step`，方便调用方复用同一套 hooks、状态迁移和工具执行逻辑。
- **可观测**：每个关键生命周期和状态迁移都提供 hook 接入点，调用方可据此实现日志、指标、流式 token 推送等能力。

## Package Structure

```
internal/core/agent/
├── doc.go                  # 包说明
├── types.go                # 核心类型：Input、Result、StopReason、Phase、State、Hooks
├── options.go              # 构造级 Option 与请求级 RunOption
├── base.go                 # Base：通用运行骨架
├── hooks.go                # NoopHooks 默认实现
├── truncate.go             # Observation 按 rune 截断
├── prompt/                 # 内置 ReAct 提示词模板
│   ├── types.go            # 模板变量类型
│   └── react_system.tmpl   # 默认系统提示词模板
└── react/                  # ReAct 主循环实现
    ├── doc.go              # 包说明
    ├── types.go            # ReAct 专有类型：Step、Decision、Planner、Parser、PromptBuilder
    ├── options.go          # Agent 构造级 Option 和请求级 RunOption
    ├── agent.go            # Agent 与 Run 主循环
    ├── parser.go           # ReActParser 文本协议解析
    ├── planner.go          # ReActTextPlanner / FunctionCallingPlanner / AutoPlanner
    ├── orchestrator.go     # 旧版兼容解析 API
    ├── prompt.go           # ReActPromptBuilder 默认提示词构造
    └── state.go            # 状态深拷贝辅助函数
```

## Architecture

### 1. 分层设计

```
┌──────────────────────────────────────────────────────┐
│  应用层（cmd/api、cmd/worker 等）                       │
│  - 注入 PromptBuilder、Hooks、系统提示词                │
│  - 提供业务工具实现                                     │
└────────────────────────┬─────────────────────────────┘
                         │ 组合
┌────────────────────────▼─────────────────────────────┐
│  core/agent/react                                     │
│  - Agent.Run() 主循环                                 │
│  - AutoPlanner 路径选择                               │
│  - ReActParser / ReActPromptBuilder                    │
└────────────────────────┬─────────────────────────────┘
                         │ 复用
┌────────────────────────▼─────────────────────────────┐
│  core/agent                                          │
│  - Base：状态持有、工具执行、hooks、错误收尾             │
│  - Phase 状态机                                       │
│  - State / Result / Input 通用类型                     │
└────────────────────────┬─────────────────────────────────┘
                         │ 依赖
┌────────────────────────▼─────────────────────────────┐
│  core/llm / core/tool                                 │
│  - llm.Client / llm.ToolCallingClient                  │
│  - tool.Registry / tool.Runner                         │
└──────────────────────────────────────────────────────┘
```

### 2. 双路径决策流程

`Agent.Run()` 每轮通过 `Planner` 生成决策，默认使用 `AutoPlanner`：

```text
state
  ├─ toolCallingEnabled && client 支持 ToolCallingClient?
  │   └─ Yes -> FunctionCallingPlanner
  │       ├─ 成功 -> Decision
  │       └─ ErrToolCallingUnsupported?
  │           └─ fallbackToReAct=true  -> 退回 ReActTextPlanner (PhaseFallback)
  └─ No -> ReActTextPlanner
      ├─ client 支持 StreamEventClient? -> planStreamTrace
      └─ No -> planTrace (同步)
```

两条路径最终都输出统一的 `Decision`：

- `DecisionFinal`：模型给出最终答案，Agent 结束运行。
- `DecisionToolCall`：模型要求调用工具，Agent 执行工具并将 `Observation` 写回下一轮。

### 3. 状态机（Phase）

Agent 每次运行会经历以下阶段迁移：

```text
PhaseStart
  -> PhaseBuildPrompt   (构造模型输入)
  -> PhaseModel         (调用模型)
  -> PhaseFallback      (仅 function calling 退回文本 ReAct 时)
  -> PhaseBuildPrompt   (重新构造文本 ReAct prompt)
  -> PhaseParse         (解析模型输出)
  -> PhaseTool          (调用工具)
  -> PhaseObserve       (记录工具观察结果)
  -> PhaseBuildPrompt   (下一轮)
  -> PhaseFinish        (得到最终答案)
  -> PhaseError         (运行因错误结束)
```

每次迁移都会触发 `BeforeTransition` 和 `AfterTransition` hooks。

### 4. 运行生命周期

`Agent.Run()` 的完整执行流程：

```text
1. Validate() 校验 base 具备运行所需依赖（client 非 nil）。
2. NewState(input) 创建初始状态，载入 registry 中的工具列表。
3. BeforeRun hook。
4. 进入迭代循环（1 到 MaxIterations）：
   a. Transition -> PhaseBuildPrompt。
   b. Transition -> PhaseModel。
   c. modelMessages() 获取即将发送给模型的消息快照。
   d. BeforeModel hook。
   e. plan() 调用 planner 生成决策。
   f. AfterModel hook（传入模型输出和错误）。
   g. 若 plan.Fallback=true：
      - Transition -> PhaseFallback。
      - Transition -> PhaseBuildPrompt。
   h. Transition -> PhaseParse。
   i. AfterParse hook。
   j. 若 decision.Kind == DecisionFinal：
      - 构造最终 Step，写入 state.Steps。
      - Transition -> PhaseFinish。
      - AfterRun hook。
      - 返回 Result。
   k. Transition -> PhaseTool。
   l. BeforeTool hook。
   m. InvokeTool() 调用工具。
   n. AfterTool hook。
   o. Transition -> PhaseObserve。
   p. 构造含 Observation 的 Step，写入 state.Steps。
   q. OnStep hook。
5. 达到 MaxIterations 仍未得到最终答案：
   - 返回 ErrMaxIterations 和部分 Steps。
```

任何步骤出错都会进入 `finishWithError`：设置 `StoppedBy`、迁移到 `PhaseError`、触发 `OnError` 和 `AfterRun` hooks。

## Public API

### 核心类型

```go
// Input 表示一次 Agent 运行的输入。
type Input struct {
    Query    string
    Messages []*llm.Message
}

// Result 表示一次 Agent 运行的结果。
type Result[S any] struct {
    Answer     string
    Steps      []S
    Iterations int
    StoppedBy  StopReason
}

// Step 表示一次迭代中的模型决策、工具调用和观察结果。
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

// Decision 表示模型输出被标准化后的结构化决策。
type Decision struct {
    Kind        DecisionKind
    Thought     string
    FinalAnswer string
    Action      string
    ActionInput coretool.Input
    ToolCallID  string
}
```

### 停止原因

```go
const (
    StopFinalAnswer   StopReason = "final_answer"   // 得到最终答案
    StopMaxIterations StopReason = "max_iterations" // 达到最大迭代次数
    StopError         StopReason = "error"          // 因错误停止
)
```

### 构造 Agent

```go
// New 创建 Agent。
// 默认检测 client 是否实现 ToolCallingClient，支持时走模型原生工具调用，否则退回文本 ReAct。
func New(client llm.Client, registry *coretool.Registry, opts ...Option) *Agent
```

### 运行 Agent

```go
// Run 执行完整 Agent 循环。
func (a *Agent) Run(ctx context.Context, input Input, opts ...RunOption) (*Result, error)
```

### 配置选项

构造级 `Option`（影响所有运行）：

| Option | 说明 |
|--------|------|
| `WithMaxIterations(n)` | 默认最大迭代次数，默认 6 |
| `WithSystemPrompt(prompt)` | 系统提示词，注入到每次构建的 prompt 中 |
| `WithPromptBuilder(builder)` | 自定义 PromptBuilder，nil 时使用默认 |
| `WithParser(parser)` | 自定义 ReAct 输出解析器 |
| `WithHooks(hooks)` | 生命周期 hooks |
| `WithModelOptions(opts...)` | 默认模型调用参数 |
| `WithObservationMaxRunes(n)` | Observation 最大 rune 数，默认 4000 |
| `WithToolRunner(runner)` | 自定义工具调用器 |
| `WithPlanner(planner)` | 自定义 Planner（跳过自动检测） |
| `WithToolCallingEnabled(enabled)` | 是否优先使用原生工具调用，默认 true |
| `WithFallbackToReAct(enabled)` | function calling 不支持时是否退回文本 ReAct，默认 true |

请求级 `RunOption`（仅影响单次运行）：

| RunOption | 说明 |
|-----------|------|
| `WithRunMaxIterations(n)` | 单次运行最大迭代次数 |
| `WithRunModelOptions(opts...)` | 单次运行模型调用参数 |

请求级配置优先于构造级配置。

## Hooks 接入点

`Hooks` 接口定义了 Agent 关键生命周期和状态迁移的钩子：

```go
type Hooks[D any, S any] interface {
    BeforeRun(ctx, state) error
    AfterRun(ctx, result, runErr) error
    BeforeTransition(ctx, state, transition) error
    AfterTransition(ctx, state, transition) error
    BeforeModel(ctx, state, messages) error
    OnToken(ctx, state, text) error             // 模型返回文本增量时
    AfterModel(ctx, state, output, modelErr) error
    AfterParse(ctx, state, decision, parseErr) error
    BeforeTool(ctx, state, call) error
    AfterTool(ctx, state, call, output, toolErr) error
    OnStep(ctx, state, step) error
    OnError(ctx, state, err) error
}
```

默认提供 `NoopHooks`，调用方可以只实现关心的方法：

```go
type myHooks struct {
    agent.NoopHooks[react.Decision, react.Step]
}

// 只覆盖需要的方法
func (h *myHooks) OnToken(ctx context.Context, state react.State, text string) error {
    // 推送流式 token 给前端
    return nil
}
```

**关于状态克隆**：所有传递给 hooks 的 `State` 都是通过 `CloneState` 生成的快照，hooks 不应依赖修改 state 来影响 Agent 内部状态。如果 `Decision` 或 `Step` 包含指针类型的字段，应通过 `SetCloneFuncs` 注册深拷贝函数，避免快照与内部状态共享可变引用。

## Planner 体系

`Planner` 负责把当前状态转成下一步标准化决策：

```go
type Planner interface {
    Plan(ctx context.Context, state State, opts ...llm.ModelCallOption) (Decision, error)
}
```

内置三种实现：

| Planner | 说明 |
|---------|------|
| `AutoPlanner` | 默认选择器，根据模型能力和配置在 function calling 与文本 ReAct 间自动选择 |
| `FunctionCallingPlanner` | 使用模型原生工具调用（需要 `llm.ToolCallingClient`） |
| `ReActTextPlanner` | 使用文本 ReAct prompt 和 parser |

### 流式文本过滤

文本 ReAct 路径中，`ReActTextPlanner` 使用 `finalAnswerEmitter` 确保只转发 `Final Answer:` 之后的文本增量到 `OnToken` hook，避免把 ReAct 协议字段（`Thought:`、`Action:` 等）泄露给前端。实现通过滑动窗口保留 marker 长度的尾部，防止 `final answer:` 标记被拆在两个流分片之间。

### Function Calling 路径

`FunctionCallingPlanner` 会把历史 `Step` 还原为模型需要的 assistant tool_call + tool result 消息对，确保原生工具调用模型能正确理解上下文。输出通过 `decisionFromToolCallingOutput` 规整为统一 `Decision`。

## Parser 与文本协议

`ReActParser` 解析基础 ReAct 文本协议：

```text
Thought: 你的思考
Action: 工具名
Action Input: {"query":"关键词"}
```

或最终答案：

```text
Thought: 我已经能回答了
Final Answer: 给用户的最终回答
```

**Action Input 解析规则**：

1. 空输入 -> 返回空 `coretool.Input`。
2. 以 `{` 开头 -> 按 JSON object 解析，解析失败返回 `ErrInvalidActionInput`。
3. 合法 JSON 但非 object（如 `[1,2]`）-> 返回 `ErrInvalidActionInput`，因为不能当作查询词静默传给工具。
4. 纯文本 -> 映射到默认字段 `query`（可通过 `WithTextActionInputKey` 自定义）。

## Prompt 内置模板

`core/agent/prompt` 使用 `embed.FS` 内嵌默认 ReAct 系统提示词模板：

```go
//go:embed *.tmpl
var Templates embed.FS
```

模板变量：

```go
type ReActSystemData struct {
    Tools        []ReActToolData  // 可用工具列表
    SystemPrompt string           // 附加系统提示词
}

type ReActToolData struct {
    Name        string
    Description string
}
```

如果默认模板不满足需求，调用方可通过 `WithPromptBuilder` 注入完全自定义的 `PromptBuilder`。

## 工具截断

`Base.TruncateObservation` 按 rune（非字节）裁剪工具观察结果，默认上限 4000 rune。裁剪在 `PhaseObserve` 阶段自动调用，避免过长 observation 撑爆后续 prompt。调用方可通过 `WithObservationMaxRunes` 调整上限；设为非正数则禁用裁剪。

## 错误处理

| 错误 | 触发场景 | StoppedBy |
|------|----------|-----------|
| `ErrMaxIterations` | 达到最大迭代次数仍未得到最终答案 | `StopMaxIterations` |
| `ErrParseDecision` | 模型输出不符合 ReAct 决策格式 | `StopError` |
| `ErrInvalidActionInput` | Action Input 不是合法 JSON object 或纯文本 | `StopError` |
| `ErrToolCallingUnsupported` | 模型不支持原生工具调用（且关闭 fallback） | `StopError` |
| `ErrStreamingUnsupported` | 模型不支持结构化流式调用 | `StopError` |

`FinishWithError` 统一收尾：设置 `LastError`、计算 `StoppedBy`、迁移到 `PhaseError`、触发 `OnError` 和 `AfterRun` hooks。

## Usage Example

```go
package main

import (
    "context"
    "fmt"

    "github.com/boxify/api-go/internal/core/agent/react"
    "github.com/boxify/api-go/internal/core/llm"
    coretool "github.com/boxify/api-go/internal/core/tool"
)

func main() {
    // 1. 准备模型客户端和工具注册表
    client := myLLMClient{} // 实现 llm.Client
    registry := coretool.NewRegistry()
    registry.Register(ctx, coretool.NewFuncTool(
        coretool.Descriptor{Name: "search", Description: "搜索知识库"},
        func(ctx context.Context, input coretool.Input) (coretool.Output, error) {
            return coretool.Output{Text: "搜索结果"}, nil
        },
    ))

    // 2. 创建 Agent
    agent := react.New(client, registry,
        react.WithSystemPrompt("你是智能助手 Cove。"),
        react.WithMaxIterations(8),
        react.WithObservationMaxRunes(2000),
        react.WithHooks(myHooks{}),
    )

    // 3. 运行
    result, err := agent.Run(ctx, react.Input{Query: "今天天气如何?"})
    if err != nil {
        // 注意：即使 err != nil，result 仍可能包含已完成的 Steps
        fmt.Printf("运行错误: %v, 停止原因: %s\n", err, result.StoppedBy)
    }
    fmt.Printf("回答: %s\n", result.Answer)
    for _, step := range result.Steps {
        fmt.Printf("迭代 %d: %s -> %s\n", step.Iteration, step.Action, step.Observation)
    }
}
```

## Implementation Structure

文件职责：

| 文件 | 职责 |
|------|------|
| `types.go` | 所有公开类型定义：`Input`、`Result`、`State`、`Hooks`、`ToolCall` 等 |
| `options.go` | 构造级 `Option[D,S]` 和请求级 `RunOption`，以及 `RunConfig` |
| `base.go` | `Base[D,S]` 结构体和各种通用方法：`NewBase`、`Transition`、`InvokeTool`、`CloneState`、`FinishWithError` |
| `hooks.go` | `NoopHooks` 空实现 |
| `truncate.go` | `truncateRunes` 工具截断辅助函数 |
| `react/types.go` | ReAct 专有类型：`Step`、`Decision`、`Planner`、`Parser`、`PromptBuilder` |
| `react/options.go` | Agent 构造级 `Option` 和请求级 `RunOption` |
| `react/agent.go` | `Agent` 结构体与 `Run()` 主循环 |
| `react/parser.go` | `ReActParser` 文本协议解析 |
| `react/planner.go` | `ReActTextPlanner`、`FunctionCallingPlanner`、`AutoPlanner` |
| `react/prompt.go` | `ReActPromptBuilder` 默认提示词构造 |
| `react/state.go` | 状态深拷贝辅助函数 |
| `prompt/types.go` | 模板变量类型 |
| `prompt/react_system.tmpl` | 默认 ReAct 系统提示词模板 |

## Failure Behavior

- **模型调用失败**：立即结束运行，`StoppedBy=StopError`，触发 `OnError` 和 `AfterRun` hooks。
- **工具调用失败**：同上，工具错误直接终止运行，不会重试。
- **达到最大迭代次数**：返回 `ErrMaxIterations` 和已完成的 `Steps`，`StoppedBy=StopMaxIterations`。
- **function calling 不支持且 fallback 开启**：自动切换到文本 ReAct，触发 `PhaseFallback` 状态迁移。
- **function calling 不支持且 fallback 关闭**：直接返回 `ErrToolCallingUnsupported`，`StoppedBy=StopError`。
- **Observation 过长**：自动按 `observationMaxRunes` 截断，不返回错误。
- **任何 hook 返回错误**：立即结束运行，hook 错误作为运行错误返回。
