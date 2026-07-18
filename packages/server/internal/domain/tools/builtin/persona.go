package builtin

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	coretool "github.com/boxify/api-go/internal/core/tool"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/repository"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
)

const maxPersonaFieldRunes = 4000

type personaField string

const (
	personaFieldSoul     personaField = "soul"
	personaFieldIdentity personaField = "identity"
)

// NewUpdateSoulTool 创建内置 update_soul 工具，整段替换当前生效角色的 Soul。
func NewUpdateSoulTool(svcCtx *svc.ServiceContext) coretool.Tool {
	return newUpdatePersonaFieldTool(svcCtx, personaFieldSoul)
}

// NewUpdateIdentityTool 创建内置 update_identity 工具，整段替换当前生效角色的 Identity。
func NewUpdateIdentityTool(svcCtx *svc.ServiceContext) coretool.Tool {
	return newUpdatePersonaFieldTool(svcCtx, personaFieldIdentity)
}

func newUpdatePersonaFieldTool(svcCtx *svc.ServiceContext, field personaField) coretool.Tool {
	return coretool.NewFuncTool(personaFieldDescriptor(field), func(ctx context.Context, input coretool.Input) (coretool.Output, error) {
		content, err := personaContentFromInput(input)
		if err != nil {
			return coretool.Output{}, err
		}
		persona, err := updateActivePersonaField(ctx, svcCtx, field, content)
		if err != nil {
			return coretool.Output{}, err
		}
		return personaFieldToolOutput(field, persona), nil
	})
}

func personaFieldDescriptor(field personaField) coretool.Descriptor {
	switch field {
	case personaFieldSoul:
		return coretool.Descriptor{
			Name: ToolUpdateSoul,
			Description: "更新当前生效角色的 Soul（性格、语气、说话风格、口头禅）。" +
				"用户要求改语气/风格，或对话中形成稳定表达偏好时调用；写入完整 Soul 正文，不要只写碎片备注。" +
				"content 为整段替换内容，可传空字符串清空。",
			Annotations: map[string]any{
				"display_name":        "更新性格灵魂",
				"display_description": "整段更新当前生效角色的语气、性格与说话风格。",
				"icon":                "✨",
				"needs_config":        false,
				"config_hint":         "",
			},
			Schema: personaContentSchema("完整的 Soul 正文：性格、语气、说话风格等。空字符串表示清空。"),
		}
	default:
		return coretool.Descriptor{
			Name: ToolUpdateIdentity,
			Description: "更新当前生效角色的 Identity（是谁、角色定位、能力边界）。" +
				"用户要求改身份/角色设定时调用；写入完整 Identity 正文，不要只写碎片备注。" +
				"content 为整段替换内容，可传空字符串清空。",
			Annotations: map[string]any{
				"display_name":        "更新身份设定",
				"display_description": "整段更新当前生效角色的身份、定位与能力边界。",
				"icon":                "🪪",
				"needs_config":        false,
				"config_hint":         "",
			},
			Schema: personaContentSchema("完整的 Identity 正文：是谁、角色定位、能力边界等。空字符串表示清空。"),
		}
	}
}

func personaContentSchema(contentDescription string) coretool.Schema {
	return coretool.Schema{
		Parameters: coretool.ParametersSchema{
			Type: "object",
			Properties: map[string]coretool.PropertySchema{
				"content": {
					"type":        "string",
					"description": contentDescription,
				},
			},
			Required:             []string{"content"},
			AdditionalProperties: false,
		},
	}
}

// personaContentFromInput 解析 content 参数；字段必须出现，允许空串清空。
func personaContentFromInput(input coretool.Input) (string, error) {
	if input == nil {
		return "", fmt.Errorf("content is required")
	}
	raw, ok := input["content"]
	if !ok || raw == nil {
		return "", fmt.Errorf("content is required")
	}
	content, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("content must be a string")
	}
	// 仅去掉首尾空白再校验长度；正文内部换行与缩进保留。
	content = strings.TrimSpace(content)
	if utf8.RuneCountInString(content) > maxPersonaFieldRunes {
		return "", xerr.BadRequest(fmt.Sprintf("content 长度不能超过 %d 个字符", maxPersonaFieldRunes))
	}
	return content, nil
}

// updateActivePersonaField 更新当前生效角色的指定人设字段。
func updateActivePersonaField(ctx context.Context, svcCtx *svc.ServiceContext, field personaField, content string) (*models.AgentPersona, error) {
	if svcCtx == nil || svcCtx.AgentPersonaRepo == nil {
		return nil, xerr.Internal("角色仓储未初始化", nil)
	}
	userID, err := util.UserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	// 与聊天注入人设一致：只改当前 is_active 角色。
	persona, err := svcCtx.AgentPersonaRepo.FindActive(ctx, userID)
	if err != nil {
		return nil, err
	}
	if persona == nil {
		return nil, xerr.BadRequest("当前没有生效角色，无法更新人设")
	}

	patch := &models.AgentPersona{}
	fields := repository.NewAgentPersonaUpdateFields()
	switch field {
	case personaFieldSoul:
		patch.Soul = content
		fields.Soul()
	case personaFieldIdentity:
		patch.Identity = content
		fields.Identity()
	default:
		return nil, xerr.Internal("未知人设字段", nil)
	}

	updated, err := svcCtx.AgentPersonaRepo.UpdateFields(ctx, userID, persona.ID, patch, fields)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func personaFieldToolOutput(field personaField, persona *models.AgentPersona) coretool.Output {
	if persona == nil {
		persona = &models.AgentPersona{}
	}
	label := "Soul"
	content := persona.Soul
	if field == personaFieldIdentity {
		label = "Identity"
		content = persona.Identity
	}
	name := strings.TrimSpace(persona.Name)
	if name == "" {
		name = persona.ID.String()
	}
	return coretool.Output{
		Text: fmt.Sprintf("已更新角色「%s」的 %s。", name, label),
		Metadata: map[string]any{
			"persona_id": persona.ID.String(),
			"field":      string(field),
			"name":       name,
			"content":    content,
		},
	}
}
