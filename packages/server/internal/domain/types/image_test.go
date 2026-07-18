package types_test

import (
	"testing"

	. "github.com/boxify/api-go/internal/domain/types"
)

// 验证图片状态常量与模型约定一致。
func TestImageStatusConstants(t *testing.T) {
	want := map[string]string{
		"pending":    ImageStatusPending,
		"processing": ImageStatusProcessing,
		"done":       ImageStatusDone,
		"failed":     ImageStatusFailed,
	}
	for name, value := range want {
		if value != name {
			t.Fatalf("status %s = %q, want %q", name, value, name)
		}
	}
}
