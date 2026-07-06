package agent

import (
	"context"
	"encoding/json"
	"strings"
)

// ReactAction 表示兼容旧接口的 ReAct 工具调用结果。
type ReactAction struct {
	Tool  string
	Input string
}

// ParseReactAction 解析 ReAct 工具调用文本。
//
// 该函数保留旧公开 API，内部委托给 ReActParser。Action Input 会按 JSON object
// 解析后重新序列化为字符串；缺失输入时返回空字符串。
func ParseReactAction(text string) (ReactAction, bool) {
	decision, err := NewReActParser().Parse(context.Background(), text)
	if err != nil || decision.Kind != DecisionToolCall {
		return ReactAction{}, false
	}
	value := ""
	if len(decision.ActionInput) > 0 {
		data, err := json.Marshal(decision.ActionInput)
		if err != nil {
			return ReactAction{}, false
		}
		value = string(data)
	}
	return ReactAction{
		Tool:  strings.TrimSpace(decision.Action),
		Input: value,
	}, true
}

// ParseReactFinal 解析 ReAct 最终答案文本。
//
// 该函数保留旧公开 API，内部委托给 ReActParser。
func ParseReactFinal(text string) (string, bool) {
	decision, err := NewReActParser().Parse(context.Background(), text)
	if err != nil || decision.Kind != DecisionFinal {
		return "", false
	}
	return strings.TrimSpace(decision.FinalAnswer), true
}
