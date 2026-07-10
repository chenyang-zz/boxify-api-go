package toolconfig

import (
	"context"
	"strings"

	coretool "github.com/boxify/api-go/internal/core/tool"
	domaintools "github.com/boxify/api-go/internal/domain/tools"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/response"
)

const builtinToolType = "builtin"

// builtinToolResponses 返回内置工具的配置响应列表
func builtinToolResponses(ctx context.Context, svcCtx *svc.ServiceContext) ([]*response.ToolConfigResponse, error) {
	catalog, err := domaintools.NewCatalog(svcCtx)
	if err != nil {
		return nil, err
	}
	registry, err := catalog.BuildRegistry(ctx, coretool.Selection{})
	if err != nil {
		return nil, err
	}
	descriptors := registry.List(nil)
	items := make([]*response.ToolConfigResponse, 0, len(descriptors))
	for _, descriptor := range descriptors {
		items = append(items, toolConfigResponseFromDescriptor(descriptor))
	}
	return items, nil
}

// toolConfigResponseFromDescriptor 将工具描述符转换为工具配置响应
func toolConfigResponseFromDescriptor(descriptor coretool.Descriptor) *response.ToolConfigResponse {
	return &response.ToolConfigResponse{
		ToolKey:     descriptor.Name,
		Name:        annotationString(descriptor.Annotations, "display_name", descriptor.Name),
		Description: annotationString(descriptor.Annotations, "display_description", descriptor.Description),
		Icon:        annotationString(descriptor.Annotations, "icon", ""),
		ToolType:    builtinToolType,
		NeedsConfig: annotationBool(descriptor.Annotations, "needs_config"),
		ConfigHit:   annotationString(descriptor.Annotations, "config_hint", ""),
		Enabled:     true,
	}
}

// annotationString 从注解中获取字符串值，如果不存在则返回默认值
func annotationString(annotations map[string]any, key string, fallback string) string {
	value, ok := annotations[key].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

// annotationBool 从注解中获取布尔值，如果不存在则返回 false
func annotationBool(annotations map[string]any, key string) bool {
	value, _ := annotations[key].(bool)
	return value
}
