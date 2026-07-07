package react_test

import (
	"testing"

	"github.com/boxify/api-go/internal/core/agent/react"
)

// 验证点：兼容函数 ParseReactAction 应复用 ReActParser 解析 JSON object 输入。
func TestParseReactAction(t *testing.T) {
	action, ok := react.ParseReactAction(`Thought: need memory
Action: memory_search
Action Input: {"query":"user preference"}`)
	if !ok {
		t.Fatal("expected action to parse")
	}
	if action.Tool != "memory_search" {
		t.Fatalf("tool = %q", action.Tool)
	}
	if action.Input != `{"query":"user preference"}` {
		t.Fatalf("input = %q", action.Input)
	}
}

// 验证点：兼容函数 ParseReactFinal 应复用 ReActParser 解析最终答案。
func TestParseReactFinalAnswer(t *testing.T) {
	final, ok := react.ParseReactFinal("Thought: enough\nFinal Answer: hello world")
	if !ok {
		t.Fatal("expected final answer")
	}
	if final != "hello world" {
		t.Fatalf("final = %q", final)
	}
}
