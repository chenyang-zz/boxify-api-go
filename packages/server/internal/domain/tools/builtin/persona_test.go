package builtin

import (
	"context"
	"reflect"
	"strings"
	"testing"

	coretool "github.com/boxify/api-go/internal/core/tool"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/google/uuid"
)

// 验证 update_soul 描述与 schema 使用 content 参数，并带展示注解。
func TestUpdateSoulDescriptor(t *testing.T) {
	tool := NewUpdateSoulTool(&svc.ServiceContext{})
	descriptor, err := tool.Describe(context.Background())
	if err != nil {
		t.Fatalf("update_soul Describe() error = %v, want nil", err)
	}
	if descriptor.Name != ToolUpdateSoul {
		t.Fatalf("update_soul Name = %q, want %q", descriptor.Name, ToolUpdateSoul)
	}
	if descriptor.Annotations["display_name"] != "更新性格灵魂" {
		t.Fatalf("update_soul display_name = %#v", descriptor.Annotations["display_name"])
	}
	wantSchema := personaContentSchema("完整的 Soul 正文：性格、语气、说话风格等。空字符串表示清空。")
	if !reflect.DeepEqual(descriptor.Schema, wantSchema) {
		t.Fatalf("update_soul Schema = %#v, want %#v", descriptor.Schema, wantSchema)
	}
}

// 验证 update_identity 描述与 schema 使用 content 参数，并带展示注解。
func TestUpdateIdentityDescriptor(t *testing.T) {
	tool := NewUpdateIdentityTool(&svc.ServiceContext{})
	descriptor, err := tool.Describe(context.Background())
	if err != nil {
		t.Fatalf("update_identity Describe() error = %v, want nil", err)
	}
	if descriptor.Name != ToolUpdateIdentity {
		t.Fatalf("update_identity Name = %q, want %q", descriptor.Name, ToolUpdateIdentity)
	}
	if descriptor.Annotations["display_name"] != "更新身份设定" {
		t.Fatalf("update_identity display_name = %#v", descriptor.Annotations["display_name"])
	}
	wantSchema := personaContentSchema("完整的 Identity 正文：是谁、角色定位、能力边界等。空字符串表示清空。")
	if !reflect.DeepEqual(descriptor.Schema, wantSchema) {
		t.Fatalf("update_identity Schema = %#v, want %#v", descriptor.Schema, wantSchema)
	}
}

// 验证 update_soul 会整段替换当前生效角色的 Soul，并返回结构化元数据。
func TestUpdateSoulReplacesActivePersonaSoul(t *testing.T) {
	userID := uuid.New()
	personaID := uuid.New()
	repo := &fakePersonaToolRepo{active: &models.AgentPersona{
		ID:       personaID,
		UserID:   userID,
		Name:     "小盒",
		Soul:     "旧性格",
		Identity: "旧身份",
		IsActive: true,
	}}
	svcCtx := &svc.ServiceContext{AgentPersonaRepo: repo}
	ctx := util.WithUserID(context.Background(), userID)

	output, err := NewUpdateSoulTool(svcCtx).Invoke(ctx, coretool.Input{"content": "  温暖、简洁  "})
	if err != nil {
		t.Fatalf("update_soul Invoke() error = %v, want nil", err)
	}
	if output.Text != "已更新角色「小盒」的 Soul。" {
		t.Fatalf("update_soul Text = %q", output.Text)
	}
	if output.Metadata["persona_id"] != personaID.String() ||
		output.Metadata["field"] != "soul" ||
		output.Metadata["content"] != "温暖、简洁" {
		t.Fatalf("update_soul metadata = %#v", output.Metadata)
	}
	if repo.active.Soul != "温暖、简洁" || repo.active.Identity != "旧身份" {
		t.Fatalf("persona after update_soul = %+v, want soul replaced only", repo.active)
	}
	if !reflect.DeepEqual(repo.updatedColumns, []string{"soul"}) {
		t.Fatalf("updated columns = %#v, want [soul]", repo.updatedColumns)
	}
}

// 验证 update_identity 会整段替换当前生效角色的 Identity。
func TestUpdateIdentityReplacesActivePersonaIdentity(t *testing.T) {
	userID := uuid.New()
	personaID := uuid.New()
	repo := &fakePersonaToolRepo{active: &models.AgentPersona{
		ID:       personaID,
		UserID:   userID,
		Name:     "小盒",
		Soul:     "旧性格",
		Identity: "旧身份",
		IsActive: true,
	}}
	svcCtx := &svc.ServiceContext{AgentPersonaRepo: repo}
	ctx := util.WithUserID(context.Background(), userID)

	output, err := NewUpdateIdentityTool(svcCtx).Invoke(ctx, coretool.Input{"content": "你是小盒助手"})
	if err != nil {
		t.Fatalf("update_identity Invoke() error = %v, want nil", err)
	}
	if output.Text != "已更新角色「小盒」的 Identity。" {
		t.Fatalf("update_identity Text = %q", output.Text)
	}
	if output.Metadata["field"] != "identity" || output.Metadata["content"] != "你是小盒助手" {
		t.Fatalf("update_identity metadata = %#v", output.Metadata)
	}
	if repo.active.Identity != "你是小盒助手" || repo.active.Soul != "旧性格" {
		t.Fatalf("persona after update_identity = %+v, want identity replaced only", repo.active)
	}
}

// 验证 content 为空字符串时清空对应字段。
func TestUpdateSoulAllowsEmptyContentToClear(t *testing.T) {
	userID := uuid.New()
	repo := &fakePersonaToolRepo{active: &models.AgentPersona{
		ID:     uuid.New(),
		UserID: userID,
		Name:   "小盒",
		Soul:   "有内容",
	}}
	ctx := util.WithUserID(context.Background(), userID)

	output, err := NewUpdateSoulTool(&svc.ServiceContext{AgentPersonaRepo: repo}).Invoke(ctx, coretool.Input{"content": ""})
	if err != nil {
		t.Fatalf("update_soul clear error = %v, want nil", err)
	}
	if repo.active.Soul != "" {
		t.Fatalf("soul after clear = %q, want empty", repo.active.Soul)
	}
	if output.Metadata["content"] != "" {
		t.Fatalf("metadata content = %#v, want empty", output.Metadata["content"])
	}
}

// 验证缺少 content 参数时返回错误。
func TestUpdatePersonaFieldRequiresContent(t *testing.T) {
	_, err := NewUpdateSoulTool(&svc.ServiceContext{}).Invoke(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "content is required") {
		t.Fatalf("missing content error = %v, want content is required", err)
	}
	_, err = NewUpdateIdentityTool(&svc.ServiceContext{}).Invoke(context.Background(), coretool.Input{})
	if err == nil || !strings.Contains(err.Error(), "content is required") {
		t.Fatalf("empty input error = %v, want content is required", err)
	}
}

// 验证 content 超过 4000 字符时拒绝。
func TestUpdatePersonaFieldRejectsTooLongContent(t *testing.T) {
	userID := uuid.New()
	repo := &fakePersonaToolRepo{active: &models.AgentPersona{ID: uuid.New(), UserID: userID, Name: "小盒"}}
	ctx := util.WithUserID(context.Background(), userID)
	long := strings.Repeat("あ", maxPersonaFieldRunes+1)

	_, err := NewUpdateSoulTool(&svc.ServiceContext{AgentPersonaRepo: repo}).Invoke(ctx, coretool.Input{"content": long})
	if err == nil || xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("too long error = %v, want bad request", err)
	}
}

// 验证没有生效角色时返回明确业务错误。
func TestUpdatePersonaFieldRequiresActivePersona(t *testing.T) {
	userID := uuid.New()
	ctx := util.WithUserID(context.Background(), userID)
	_, err := NewUpdateSoulTool(&svc.ServiceContext{AgentPersonaRepo: &fakePersonaToolRepo{}}).Invoke(ctx, coretool.Input{"content": "x"})
	if err == nil || xerr.From(err).Kind != xerr.KindBadRequest {
		t.Fatalf("no active persona error = %v, want bad request", err)
	}
}

// 验证角色仓储未初始化时返回内部错误。
func TestUpdatePersonaFieldRequiresRepo(t *testing.T) {
	userID := uuid.New()
	ctx := util.WithUserID(context.Background(), userID)
	_, err := NewUpdateIdentityTool(&svc.ServiceContext{}).Invoke(ctx, coretool.Input{"content": "x"})
	if err == nil || xerr.From(err).Kind != xerr.KindInternal {
		t.Fatalf("nil repo error = %v, want internal", err)
	}
}

// 验证 context 缺少 user 时返回未登录错误。
func TestUpdatePersonaFieldRequiresUser(t *testing.T) {
	repo := &fakePersonaToolRepo{active: &models.AgentPersona{ID: uuid.New(), Name: "小盒"}}
	_, err := NewUpdateSoulTool(&svc.ServiceContext{AgentPersonaRepo: repo}).Invoke(context.Background(), coretool.Input{"content": "x"})
	if err == nil || xerr.From(err).Kind != xerr.KindUnauthorized {
		t.Fatalf("missing user error = %v, want unauthorized", err)
	}
}

type fakePersonaToolRepo struct {
	active          *models.AgentPersona
	updatedColumns  []string
	findActiveErr   error
	updateFieldsErr error
}

func (r *fakePersonaToolRepo) Create(_ context.Context, _ uuid.UUID, row *models.AgentPersona) (*models.AgentPersona, error) {
	return row, nil
}
func (r *fakePersonaToolRepo) List(_ context.Context, _ uuid.UUID) ([]*models.AgentPersona, error) {
	return nil, nil
}
func (r *fakePersonaToolRepo) FindByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*models.AgentPersona, error) {
	return nil, xerr.NotFound("智能体人格不存在")
}
func (r *fakePersonaToolRepo) FindActive(_ context.Context, _ uuid.UUID) (*models.AgentPersona, error) {
	if r.findActiveErr != nil {
		return nil, r.findActiveErr
	}
	if r.active == nil {
		return nil, nil
	}
	// 返回副本，避免测试误改调用方状态。
	clone := *r.active
	return &clone, nil
}
func (r *fakePersonaToolRepo) Update(_ context.Context, _ uuid.UUID, row *models.AgentPersona) (*models.AgentPersona, error) {
	return row, nil
}
func (r *fakePersonaToolRepo) UpdateFields(_ context.Context, _ uuid.UUID, agentPersonaID uuid.UUID, patch *models.AgentPersona, fields *repository.AgentPersonaUpdateFields) (*models.AgentPersona, error) {
	if r.updateFieldsErr != nil {
		return nil, r.updateFieldsErr
	}
	if r.active == nil || r.active.ID != agentPersonaID {
		return nil, xerr.NotFound("智能体人格不存在")
	}
	r.updatedColumns = fields.Columns()
	for _, col := range r.updatedColumns {
		switch col {
		case "soul":
			r.active.Soul = patch.Soul
		case "identity":
			r.active.Identity = patch.Identity
		}
	}
	clone := *r.active
	return &clone, nil
}
func (r *fakePersonaToolRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (r *fakePersonaToolRepo) Count(_ context.Context, _ uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *fakePersonaToolRepo) ActivateByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
