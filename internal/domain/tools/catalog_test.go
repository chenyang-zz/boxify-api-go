package tools

import (
	"context"
	"reflect"
	"testing"
	"time"

	coretool "github.com/boxify/api-go/internal/core/tool"
)

// 验证 NewCatalog 会注册 system 工具集，作为领域层基础工具的统一入口。
func TestNewCatalogRegistersSystemToolSet(t *testing.T) {
	ctx := context.Background()

	catalog, err := NewCatalog()
	if err != nil {
		t.Fatalf("NewCatalog() error = %v, want nil", err)
	}
	sets, err := catalog.ListSets(ctx)
	if err != nil {
		t.Fatalf("Catalog.ListSets() error = %v, want nil", err)
	}

	gotNames := make([]string, 0, len(sets))
	for _, set := range sets {
		gotNames = append(gotNames, set.Name)
	}
	wantNames := []string{ToolSetSystem}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("Catalog.ListSets() names = %#v, want %#v", gotNames, wantNames)
	}
}

// 验证 NewCatalog 展开的注册表可以按名称调用 current_time 工具。
func TestNewCatalogBuildRegistryInvokesCurrentTime(t *testing.T) {
	ctx := context.Background()
	fixed := mustParseTime(t, "2026-07-05T10:11:12Z")
	catalog, err := NewCatalog(WithClock(func() time.Time {
		return fixed
	}))
	if err != nil {
		t.Fatalf("NewCatalog() error = %v, want nil", err)
	}

	registry, err := catalog.BuildRegistry(ctx, coretool.Selection{ToolNames: []string{ToolCurrentTime}})
	if err != nil {
		t.Fatalf("Catalog.BuildRegistry(current_time) error = %v, want nil", err)
	}
	output, err := coretool.NewRunner(registry).Invoke(ctx, ToolCurrentTime, nil)
	if err != nil {
		t.Fatalf("Runner.Invoke(current_time) error = %v, want nil", err)
	}
	if output.Text != "2026-07-05T10:11:12Z" {
		t.Fatalf("Runner.Invoke(current_time).Text = %q, want 2026-07-05T10:11:12Z", output.Text)
	}
}
