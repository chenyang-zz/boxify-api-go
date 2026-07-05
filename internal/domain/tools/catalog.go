package tools

import (
	"context"
	"fmt"

	coretool "github.com/boxify/api-go/internal/core/tool"
)

const (
	// ToolSetSystem 是领域层系统工具集名称。
	ToolSetSystem = "system"
	// ToolCurrentTime 是获取当前时间的工具名称。
	ToolCurrentTime = "current_time"
)

// NewCatalog 创建并返回领域层本地工具目录。
//
// 返回的 Catalog 当前包含 system 工具集，其中注册 current_time。opts 会应用到
// 所有需要长期配置的领域工具；注册工具集失败时返回错误。
func NewCatalog(opts ...Option) (*coretool.Catalog, error) {
	cfg := applyOptions(opts...)
	catalog := coretool.NewCatalog()
	systemSet := coretool.NewStaticSet(coretool.SetDescriptor{
		Name:        ToolSetSystem,
		Description: "System tools for runtime context.",
		Tags:        []string{"system"},
	}, newCurrentTimeTool(cfg))
	if err := catalog.RegisterSet(context.Background(), systemSet); err != nil {
		return nil, fmt.Errorf("register system tool set: %w", err)
	}
	return catalog, nil
}
