package tool

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// 验证点：函数工具应返回注册时的描述，并把输入交给调用函数处理。
func TestFuncToolDescribesAndInvokes(t *testing.T) {
	ctx := context.Background()
	desc := Descriptor{
		Name:        "search",
		Description: "Search documents.",
		Schema: Schema{
			Parameters: ParametersSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"query": {
						"type":        "string",
						"description": "Search query.",
					},
				},
				Required:             []string{"query"},
				AdditionalProperties: false,
			},
		},
	}
	ft := NewFuncTool(desc, func(ctx context.Context, input Input) (Output, error) {
		return Output{
			Text: "query=" + input["query"].(string),
			Metadata: map[string]any{
				"count": 1,
			},
		}, nil
	})

	gotDesc, err := ft.Describe(ctx)
	if err != nil {
		t.Fatalf("FuncTool.Describe() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(gotDesc, desc) {
		t.Fatalf("FuncTool.Describe() = %#v, want %#v", gotDesc, desc)
	}

	got, err := ft.Invoke(ctx, Input{"query": "rag"})
	if err != nil {
		t.Fatalf("FuncTool.Invoke() error = %v, want nil", err)
	}
	if got.Text != "query=rag" {
		t.Fatalf("FuncTool.Invoke().Text = %q, want %q", got.Text, "query=rag")
	}
	if got.Metadata["count"] != 1 {
		t.Fatalf("FuncTool.Invoke().Metadata[count] = %#v, want 1", got.Metadata["count"])
	}
}

// 验证点：CloneDescriptor 应复制 schema、required、annotations 和 strict 指针，避免调用方修改原始描述污染副本。
func TestCloneDescriptorReturnsIndependentCopy(t *testing.T) {
	strict := true
	original := Descriptor{
		Name:        "search",
		Description: "Search documents.",
		Schema: Schema{
			Parameters: ParametersSchema{
				Type: "object",
				Properties: map[string]PropertySchema{
					"query": {"type": "string"},
				},
				Required:             []string{"query"},
				AdditionalProperties: map[string]any{"type": "never"},
			},
			Strict: &strict,
		},
		Annotations: map[string]any{"scope": "rag"},
	}

	cloned := CloneDescriptor(original)
	original.Schema.Parameters.Properties["query"]["type"] = "number"
	original.Schema.Parameters.Required[0] = "changed"
	original.Schema.Parameters.AdditionalProperties.(map[string]any)["type"] = "changed"
	original.Annotations["scope"] = "changed"
	*original.Schema.Strict = false

	if cloned.Schema.Parameters.Properties["query"]["type"] != "string" {
		t.Fatalf("CloneDescriptor().Schema.Parameters.Properties[query][type] = %#v, want string", cloned.Schema.Parameters.Properties["query"]["type"])
	}
	if cloned.Schema.Parameters.Required[0] != "query" {
		t.Fatalf("CloneDescriptor().Schema.Parameters.Required[0] = %q, want query", cloned.Schema.Parameters.Required[0])
	}
	if cloned.Schema.Parameters.AdditionalProperties.(map[string]any)["type"] != "never" {
		t.Fatalf("CloneDescriptor().Schema.Parameters.AdditionalProperties[type] = %#v, want never", cloned.Schema.Parameters.AdditionalProperties.(map[string]any)["type"])
	}
	if cloned.Annotations["scope"] != "rag" {
		t.Fatalf("CloneDescriptor().Annotations[scope] = %#v, want rag", cloned.Annotations["scope"])
	}
	if cloned.Schema.Strict == original.Schema.Strict || cloned.Schema.Strict == nil || !*cloned.Schema.Strict {
		t.Fatalf("CloneDescriptor().Schema.Strict = %#v, want independent true pointer", cloned.Schema.Strict)
	}
}

// 验证点：CloneDescriptors 应返回独立切片，并逐个复制 descriptor 内部字段。
func TestCloneDescriptorsReturnsIndependentSlice(t *testing.T) {
	original := []Descriptor{
		{
			Name: "alpha",
			Schema: Schema{Parameters: ParametersSchema{
				Properties: map[string]PropertySchema{"query": {"type": "string"}},
			}},
		},
	}

	cloned := CloneDescriptors(original)
	original[0].Name = "changed"
	original[0].Schema.Parameters.Properties["query"]["type"] = "number"

	if len(cloned) != 1 {
		t.Fatalf("CloneDescriptors() len = %d, want 1", len(cloned))
	}
	if cloned[0].Name != "alpha" {
		t.Fatalf("CloneDescriptors()[0].Name = %q, want alpha", cloned[0].Name)
	}
	if cloned[0].Schema.Parameters.Properties["query"]["type"] != "string" {
		t.Fatalf("CloneDescriptors()[0].Schema.Parameters.Properties[query][type] = %#v, want string", cloned[0].Schema.Parameters.Properties["query"]["type"])
	}
}

// 验证点：注册表应拒绝 nil 工具、空名称工具和重复名称工具，避免运行期歧义。
func TestRegistryRejectsInvalidTools(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	if err := registry.Register(ctx, nil); err == nil {
		t.Fatalf("Registry.Register(nil) error = nil, want error")
	}

	emptyName := NewFuncTool(Descriptor{Name: " "}, func(ctx context.Context, input Input) (Output, error) {
		return Output{}, nil
	})
	if err := registry.Register(ctx, emptyName); err == nil {
		t.Fatalf("Registry.Register(empty name) error = nil, want error")
	}

	first := NewFuncTool(Descriptor{Name: "lookup"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{Text: "first"}, nil
	})
	if err := registry.Register(ctx, first); err != nil {
		t.Fatalf("Registry.Register(first) error = %v, want nil", err)
	}

	duplicate := NewFuncTool(Descriptor{Name: "lookup"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{Text: "duplicate"}, nil
	})
	if err := registry.Register(ctx, duplicate); err == nil {
		t.Fatalf("Registry.Register(duplicate) error = nil, want error")
	}
}

// 验证点：工具清单应按名称稳定排序，并支持 enabled 过滤。
func TestRegistryListReturnsStableDescriptors(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	for _, name := range []string{"zeta", "alpha", "beta"} {
		toolName := name
		err := registry.Register(ctx, NewFuncTool(Descriptor{Name: toolName}, func(ctx context.Context, input Input) (Output, error) {
			return Output{Text: toolName}, nil
		}))
		if err != nil {
			t.Fatalf("Registry.Register(%q) error = %v, want nil", toolName, err)
		}
	}

	got := registry.List(nil)
	gotNames := descriptorNames(got)
	wantNames := []string{"alpha", "beta", "zeta"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("Registry.List(nil) names = %#v, want %#v", gotNames, wantNames)
	}

	got = registry.List(map[string]bool{"beta": true, "missing": true})
	gotNames = descriptorNames(got)
	wantNames = []string{"beta"}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("Registry.List(enabled) names = %#v, want %#v", gotNames, wantNames)
	}
}

// 验证点：Runner 应按工具名查找并调用工具，nil input 应被规整为空输入。
func TestRunnerInvokeCallsRegisteredTool(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	err := registry.Register(ctx, NewFuncTool(Descriptor{Name: "echo"}, func(ctx context.Context, input Input) (Output, error) {
		if input == nil {
			t.Fatalf("FuncTool input = nil, want empty map")
		}
		return Output{Text: "ok"}, nil
	}))
	if err != nil {
		t.Fatalf("Registry.Register(echo) error = %v, want nil", err)
	}

	got, err := NewRunner(registry).Invoke(ctx, "echo", nil)
	if err != nil {
		t.Fatalf("Runner.Invoke(echo) error = %v, want nil", err)
	}
	if got.Text != "ok" {
		t.Fatalf("Runner.Invoke(echo).Text = %q, want %q", got.Text, "ok")
	}
}

// 验证点：默认策略应把未知工具错误转成可观察输出，方便模型后续修正。
func TestRunnerInvokeUnknownToolReturnsErrorOutputByDefault(t *testing.T) {
	got, err := NewRunner(NewRegistry()).Invoke(context.Background(), "missing", Input{})
	if err != nil {
		t.Fatalf("Runner.Invoke(missing) error = %v, want nil", err)
	}
	if got.Text == "" {
		t.Fatalf("Runner.Invoke(missing).Text = empty, want error text")
	}
	if got.Metadata["error"] == nil {
		t.Fatalf("Runner.Invoke(missing).Metadata[error] = nil, want error metadata")
	}
}

// 验证点：严格策略下工具执行错误应直接返回 error，不包装成输出。
func TestRunnerInvokeReturnsErrorWhenErrorAsOutputDisabled(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	invokeErr := errors.New("backend unavailable")
	err := registry.Register(ctx, NewFuncTool(Descriptor{Name: "fail"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{}, invokeErr
	}))
	if err != nil {
		t.Fatalf("Registry.Register(fail) error = %v, want nil", err)
	}

	got, err := NewRunner(registry, WithErrorAsOutput(false)).Invoke(ctx, "fail", Input{})
	if !errors.Is(err, invokeErr) {
		t.Fatalf("Runner.Invoke(fail) error = %v, want %v", err, invokeErr)
	}
	if !reflect.DeepEqual(got, Output{}) {
		t.Fatalf("Runner.Invoke(fail) output = %#v, want zero Output", got)
	}
}

// TestRunnerInvokePreservesOutputWhenToolReturnsError 验证默认 Runner 保留错误携带的完整输出，严格模式则只返回原始错误。
func TestRunnerInvokePreservesOutputWhenToolReturnsError(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	want := Output{
		Text:     "remote rejected request",
		Parts:    []Part{{Type: "image", Data: []byte{1, 2, 3}, MIME: "image/png"}},
		Metadata: map[string]any{"mcp_is_error": true, "structured_content": map[string]any{"code": "bad_request"}},
	}
	invokeErr := errors.New("mcp tool returned error")
	err := registry.Register(ctx, NewFuncTool(Descriptor{Name: "remote"}, func(context.Context, Input) (Output, error) {
		return want, invokeErr
	}))
	if err != nil {
		t.Fatalf("Registry.Register(remote) error = %v, want nil", err)
	}

	got, err := NewRunner(registry).Invoke(ctx, "remote", nil)
	if err != nil {
		t.Fatalf("Runner.Invoke(remote) error = %v, want nil", err)
	}
	if got.Text != "tool invocation failed:\n"+want.Text || len(got.Parts) != 1 || !reflect.DeepEqual(got.Parts[0].Data, want.Parts[0].Data) {
		t.Fatalf("Runner.Invoke(remote) output = %#v, want preserved text and parts", got)
	}
	if got.Metadata["mcp_is_error"] != true || got.Metadata["error"] != invokeErr.Error() {
		t.Fatalf("Runner.Invoke(remote) metadata = %#v, want preserved MCP marker and standard error", got.Metadata)
	}

	strictOutput, strictErr := NewRunner(registry, WithErrorAsOutput(false)).Invoke(ctx, "remote", nil)
	if !errors.Is(strictErr, invokeErr) {
		t.Fatalf("strict Runner.Invoke(remote) error = %v, want %v", strictErr, invokeErr)
	}
	if !reflect.DeepEqual(strictOutput, Output{}) {
		t.Fatalf("strict Runner.Invoke(remote) output = %#v, want zero Output", strictOutput)
	}

	emptyErr := errors.New("backend unavailable")
	err = registry.Register(ctx, NewFuncTool(Descriptor{Name: "empty"}, func(context.Context, Input) (Output, error) {
		return Output{}, emptyErr
	}))
	if err != nil {
		t.Fatalf("Registry.Register(empty) error = %v, want nil", err)
	}
	emptyOutput, err := NewRunner(registry).Invoke(ctx, "empty", nil)
	if err != nil || emptyOutput.Text != "tool invocation failed:\n"+emptyErr.Error() {
		t.Fatalf("Runner.Invoke(empty) output/error = %#v/%v, want prefixed fallback text", emptyOutput, err)
	}
}

// 验证点：工具输出应保留结构化 part 和 metadata，便于后续扩展图片或文件结果。
func TestOutputCarriesPartsAndMetadata(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()
	err := registry.Register(ctx, NewFuncTool(Descriptor{Name: "image"}, func(ctx context.Context, input Input) (Output, error) {
		return Output{
			Text: "done",
			Parts: []Part{
				{Type: "image", Data: []byte{1, 2, 3}, MIME: "image/png"},
				{Type: "text", Text: "caption"},
			},
			Metadata: map[string]any{"source": "unit-test"},
		}, nil
	}))
	if err != nil {
		t.Fatalf("Registry.Register(image) error = %v, want nil", err)
	}

	got, err := NewRunner(registry).Invoke(ctx, "image", Input{})
	if err != nil {
		t.Fatalf("Runner.Invoke(image) error = %v, want nil", err)
	}
	if len(got.Parts) != 2 {
		t.Fatalf("Runner.Invoke(image) parts len = %d, want 2", len(got.Parts))
	}
	if got.Parts[0].MIME != "image/png" || !reflect.DeepEqual(got.Parts[0].Data, []byte{1, 2, 3}) {
		t.Fatalf("Runner.Invoke(image) first part = %#v, want png bytes", got.Parts[0])
	}
	if got.Metadata["source"] != "unit-test" {
		t.Fatalf("Runner.Invoke(image).Metadata[source] = %#v, want unit-test", got.Metadata["source"])
	}
}

// TestParametersSchemaMapPreservesRawFields 验证完整 JSON Schema 扩展字段会保留，且强类型字段具有覆盖优先级。
func TestParametersSchemaMapPreservesRawFields(t *testing.T) {
	schema := NewParametersSchema(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{"type": "string"},
		},
		"required": []any{"query"},
		"$defs": map[string]any{
			"filter": map[string]any{"type": "object"},
		},
		"oneOf": []any{map[string]any{"required": []any{"query"}}},
	})
	schema.Type = "custom"

	got := schema.Map()
	if got["type"] != "custom" {
		t.Fatalf("ParametersSchema.Map type = %#v, want custom", got["type"])
	}
	if got["$defs"] == nil || got["oneOf"] == nil {
		t.Fatalf("ParametersSchema.Map = %#v, want raw $defs and oneOf", got)
	}
	if len(schema.Required) != 1 || schema.Required[0] != "query" {
		t.Fatalf("NewParametersSchema required = %#v, want query", schema.Required)
	}

	cloned := CloneDescriptor(Descriptor{Schema: Schema{Parameters: schema}})
	schema.Raw["$defs"] = nil
	if cloned.Schema.Parameters.Raw["$defs"] == nil {
		t.Fatal("CloneDescriptor raw $defs = nil, want independent top-level map")
	}
}

func descriptorNames(descriptors []Descriptor) []string {
	names := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		names = append(names, descriptor.Name)
	}
	return names
}
