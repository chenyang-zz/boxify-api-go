package types

import "testing"

// 验证标签 scope 常量集中由 domain 类型包提供，避免 logic/repository 重复定义字符串。
func TestTagScopeConstants(t *testing.T) {
	tests := map[string]TagScope{
		"all":      TagScopeAll,
		"document": TagScopeDocument,
		"image":    TagScopeImage,
	}
	for want, got := range tests {
		if string(got) != want {
			t.Fatalf("tag scope = %q, want %q", got, want)
		}
	}
}
