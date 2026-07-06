// Package agent 提供业务无关的 Agent 主循环。
//
// 本包优先使用模型原生 function/tool calling 能力生成工具调用决策；当模型客户端
// 不支持该能力，或调用方显式关闭时，会退回到文本 ReAct prompt 与 parser。两条路径
// 都会被规整为统一的 Decision 和 Step，方便调用方复用同一套 hooks、状态迁移和工具
// 执行逻辑。
package agent
