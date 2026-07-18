package tool

import (
	"context"
	"encoding/json"
)

// Schema 表示模型或协议用于理解如何调用工具的 schema。
//
// Parameters 描述工具参数的顶层 JSON Schema。Strict 供 OpenAI 等支持严格结构化
// 输出的适配器使用；core/tool 只负责保存和传递该配置。
type Schema struct {
	Parameters ParametersSchema `json:"parameters,omitempty"`
	Strict     *bool            `json:"strict,omitempty"`
}

// ParametersSchema 描述工具参数对象的顶层 JSON Schema。
//
// Properties 的单个字段 schema 仍保持开放 map，便于表达 enum、items、oneOf、
// default 等完整 JSON Schema 能力。
type ParametersSchema struct {
	Raw                  map[string]any            `json:"-"`
	Type                 string                    `json:"type,omitempty"`
	Properties           map[string]PropertySchema `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AdditionalProperties any                       `json:"additionalProperties,omitempty"`
}

// NewParametersSchema 从完整 JSON Schema 构建参数结构。
//
// 返回值会保留所有顶层字段到 Raw，同时投影 type、properties、required 和
// additionalProperties，供现有模型适配器继续使用强类型字段。nil 输入返回默认对象 schema。
func NewParametersSchema(raw map[string]any) ParametersSchema {
	schema := ParametersSchema{Raw: cloneMap(raw)}
	if value, ok := raw["type"].(string); ok {
		schema.Type = value
	}
	if values, ok := raw["properties"].(map[string]any); ok {
		schema.Properties = make(map[string]PropertySchema, len(values))
		for key, value := range values {
			if property, ok := value.(map[string]any); ok {
				schema.Properties[key] = PropertySchema(cloneMap(property))
			}
		}
	}
	if values, ok := raw["required"].([]any); ok {
		for _, value := range values {
			if item, ok := value.(string); ok {
				schema.Required = append(schema.Required, item)
			}
		}
	} else if values, ok := raw["required"].([]string); ok {
		schema.Required = cloneStrings(values)
	}
	if value, ok := raw["additionalProperties"]; ok {
		schema.AdditionalProperties = cloneAny(value)
	}
	return schema
}

// Map 返回可直接发送给模型供应商的完整 JSON Schema 副本。
//
// Raw 中的扩展字段会保留，强类型字段具有更高优先级；缺少 type 时默认补为 object。
func (s ParametersSchema) Map() map[string]any {
	out := cloneMap(s.Raw)
	if out == nil {
		out = map[string]any{}
	}
	if s.Type != "" {
		out["type"] = s.Type
	} else if _, ok := out["type"]; !ok {
		out["type"] = "object"
	}
	if len(s.Properties) > 0 {
		out["properties"] = s.Properties
	}
	if len(s.Required) > 0 {
		out["required"] = s.Required
	}
	if s.AdditionalProperties != nil {
		out["additionalProperties"] = s.AdditionalProperties
	}
	return out
}

// MarshalJSON 将参数结构编码为完整 JSON Schema。
func (s ParametersSchema) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Map())
}

// PropertySchema 描述单个参数字段的 JSON Schema。
type PropertySchema map[string]any

// Descriptor 描述一个可以暴露给模型或编排器选择的工具。
//
// Name 必须非空，并在同一个 Registry 中唯一。Schema 表示调用 schema，
// Annotations 用于承载模型或 UI 可选理解的附加元数据。
type Descriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      Schema         `json:"schema,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// SetDescriptor 描述一个工具集。
//
// Name 必须非空，并在同一个 Catalog 中唯一。Tags 可用于调用方做业务侧筛选，
// Annotations 用于承载 UI 或模型提示所需的附加元数据。
type SetDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

// Input 表示传给工具的通用结构化参数。
type Input map[string]any

// Output 表示工具调用结果。
//
// Text 是最常见的文本观察结果。Parts 用于表达图片、文件或多段文本等结构化结果。
// Metadata 用于返回调用统计、错误类型或调用方需要保留的非展示字段。
type Output struct {
	Text     string         `json:"text,omitempty"`
	Parts    []Part         `json:"parts,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Part 表示工具输出中的一个结构化片段。
//
// Type 由调用方约定，常见值可以是 text、image、file。Text 用于文本片段，
// Data 用于二进制片段，MIME 描述 Data 的媒体类型。
type Part struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
	Data []byte `json:"data,omitempty"`
	MIME string `json:"mime,omitempty"`
}

// Tool 表示一个可被模型或编排器调用的工具。
//
// Describe 返回稳定的工具元信息。Invoke 执行工具调用；实现应尊重 ctx 的取消信号，
// 并把业务依赖通过构造函数、闭包或 ctx 注入，而不是依赖全局状态。
type Tool interface {
	Describe(ctx context.Context) (Descriptor, error)
	Invoke(ctx context.Context, input Input) (Output, error)
}

// ToolSet 表示一组可展开成 Tool 的工具集合。
//
// ToolSet 只负责组织工具，不负责执行工具。调用方可以把 ToolSet 注册到 Catalog，
// 再通过 Catalog.BuildRegistry 生成扁平 Registry 交给 Runner 调用。
type ToolSet interface {
	Describe(ctx context.Context) (SetDescriptor, error)
	Tools(ctx context.Context) ([]Tool, error)
}

// InvokeFunc 是 FuncTool 使用的函数签名。
type InvokeFunc func(ctx context.Context, input Input) (Output, error)

// Selection 描述从 Catalog 中选择哪些工具集和工具。
//
// SetNames 为空表示不限制工具集。ToolNames 为空表示不限制工具。两个字段都为空时，
// Catalog.BuildRegistry 会展开全部工具集和全部工具。
type Selection struct {
	SetNames  []string
	ToolNames []string
}
