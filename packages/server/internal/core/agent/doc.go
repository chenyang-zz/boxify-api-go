// Package agent 提供业务无关的 Agent 运行骨架。
//
// 本包只承载通用状态、生命周期 hooks、工具执行和错误收尾能力；具体的模型决策协议、
// prompt 构造和 parser 由子包实现，例如 agent/react。
package agent
