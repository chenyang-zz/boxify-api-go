package toolconfig

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// 验证工具配置列表来自内置 Catalog，且没有数据库配置时默认启用并忽略未知历史记录。
func TestListToolConfigsUsesBuiltinCatalogAndDefaultsEnabled(t *testing.T) {
	repo := &fakeToolConfigRepo{rows: []*models.ToolConfig{{ID: uuid.New(), ToolKey: "removed_tool", Enabled: false}}}
	logic := NewListToolConfigsLogic(context.Background(), &svc.ServiceContext{ToolConfigRepo: repo})

	out, err := logic.ListToolConfigs(uuid.New())
	if err != nil {
		t.Fatalf("ListToolConfigs error = %v, want nil", err)
	}
	if len(out.List) != 2 {
		t.Fatalf("ListToolConfigs list len = %d, want 2", len(out.List))
	}
	keys := []string{out.List[0].ToolKey, out.List[1].ToolKey}
	if !slices.Equal(keys, []string{"current_time", "knowledge_search"}) {
		t.Fatalf("ListToolConfigs keys = %#v, want builtin catalog keys", keys)
	}
	for _, item := range out.List {
		if !item.Enabled || item.ToolType != builtinToolType || item.Name == "" || item.Description == "" || item.Icon == "" {
			t.Fatalf("ListToolConfigs item = %+v, want enabled builtin display metadata", item)
		}
	}
}

// 验证数据库中最新的用户配置会覆盖内置工具的默认启用状态。
func TestListToolConfigsAppliesLatestPersistedState(t *testing.T) {
	repo := &fakeToolConfigRepo{rows: []*models.ToolConfig{
		{ID: uuid.New(), ToolKey: "current_time", Enabled: false},
		{ID: uuid.New(), ToolKey: "current_time", Enabled: true},
	}}
	logic := NewListToolConfigsLogic(context.Background(), &svc.ServiceContext{ToolConfigRepo: repo})

	out, err := logic.ListToolConfigs(uuid.New())
	if err != nil {
		t.Fatalf("ListToolConfigs error = %v, want nil", err)
	}
	if out.List[0].ToolKey != "current_time" || out.List[0].Enabled {
		t.Fatalf("current_time = %+v, want latest persisted disabled state", out.List[0])
	}
}

// 验证关闭尚未持久化的内置工具会创建用户配置，并正确保留 false 值。
func TestToggleToolCreatesPersistedStateForBuiltinTool(t *testing.T) {
	repo := &fakeToolConfigRepo{}
	logic := NewToggleToolLogic(context.Background(), &svc.ServiceContext{ToolConfigRepo: repo})
	enabled := false

	err := logic.ToggleTool(uuid.New(), &request.ToggleToolRequest{
		UriToolKeyRequest: request.UriToolKeyRequest{ToolKey: " current_time "},
		Enabled:           &enabled,
	})
	if err != nil {
		t.Fatalf("ToggleTool error = %v, want nil", err)
	}
	if repo.created == nil || repo.created.ToolKey != "current_time" || repo.created.ToolType != builtinToolType || repo.created.Enabled {
		t.Fatalf("created = %+v, want disabled builtin config", repo.created)
	}
	if repo.created.ID == uuid.Nil {
		t.Fatal("created ID = nil UUID, want generated ID")
	}
}

// 验证已有工具配置只更新 enabled 字段，不创建重复记录。
func TestToggleToolUpdatesOnlyEnabledForExistingConfig(t *testing.T) {
	configID := uuid.New()
	repo := &fakeToolConfigRepo{rows: []*models.ToolConfig{{ID: configID, ToolKey: "knowledge_search", Enabled: false}}}
	logic := NewToggleToolLogic(context.Background(), &svc.ServiceContext{ToolConfigRepo: repo})
	enabled := true

	err := logic.ToggleTool(uuid.New(), &request.ToggleToolRequest{
		UriToolKeyRequest: request.UriToolKeyRequest{ToolKey: "knowledge_search"},
		Enabled:           &enabled,
	})
	if err != nil {
		t.Fatalf("ToggleTool error = %v, want nil", err)
	}
	if repo.created != nil || repo.updatedID != configID || repo.updated == nil || !repo.updated.Enabled {
		t.Fatalf("toggle calls created=%+v updatedID=%s updated=%+v", repo.created, repo.updatedID, repo.updated)
	}
	if !slices.Equal(repo.updatedFields.Columns(), []string{"enabled"}) {
		t.Fatalf("updated fields = %#v, want enabled only", repo.updatedFields.Columns())
	}
}

// 验证未知工具无法创建配置。
func TestToggleToolRejectsUnknownTool(t *testing.T) {
	repo := &fakeToolConfigRepo{}
	logic := NewToggleToolLogic(context.Background(), &svc.ServiceContext{ToolConfigRepo: repo})
	enabled := true

	err := logic.ToggleTool(uuid.New(), &request.ToggleToolRequest{
		UriToolKeyRequest: request.UriToolKeyRequest{ToolKey: "missing"},
		Enabled:           &enabled,
	})
	if xerr.From(err).Kind != xerr.KindNotFound {
		t.Fatalf("ToggleTool error = %v, want not found", err)
	}
	if repo.created != nil {
		t.Fatalf("created = %+v, want nil", repo.created)
	}
}

type fakeToolConfigRepo struct {
	rows          []*models.ToolConfig
	listErr       error
	created       *models.ToolConfig
	updatedID     uuid.UUID
	updated       *models.ToolConfig
	updatedFields *repository.ToolConfigUpdateFields
}

func (r *fakeToolConfigRepo) Create(_ context.Context, userID uuid.UUID, row *models.ToolConfig) (*models.ToolConfig, error) {
	row.UserID = userID
	r.created = row
	r.rows = append([]*models.ToolConfig{row}, r.rows...)
	return row, nil
}

func (r *fakeToolConfigRepo) List(_ context.Context, _ uuid.UUID) ([]*models.ToolConfig, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return append([]*models.ToolConfig(nil), r.rows...), nil
}

func (r *fakeToolConfigRepo) FindByID(_ context.Context, _ uuid.UUID, id uuid.UUID) (*models.ToolConfig, error) {
	for _, row := range r.rows {
		if row != nil && row.ID == id {
			return row, nil
		}
	}
	return nil, xerr.NotFound("工具配置不存在")
}

func (r *fakeToolConfigRepo) Update(_ context.Context, _ uuid.UUID, row *models.ToolConfig) (*models.ToolConfig, error) {
	return row, nil
}

func (r *fakeToolConfigRepo) UpdateFields(_ context.Context, _ uuid.UUID, id uuid.UUID, row *models.ToolConfig, fields *repository.ToolConfigUpdateFields) (*models.ToolConfig, error) {
	r.updatedID = id
	r.updated = row
	r.updatedFields = fields
	return row, nil
}

func (r *fakeToolConfigRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return errors.New("not implemented")
}
