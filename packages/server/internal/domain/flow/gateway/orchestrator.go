// Package gateway 编排消息网关的数据面业务流程。
package gateway

import (
	"log/slog"

	corechannel "github.com/boxify/api-go/internal/core/channel"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/xerr"
)

// Orchestrator 协调网关入站、媒体处理、状态恢复和账号运行配置。
type Orchestrator struct {
	svc *svc.ServiceContext
	log *slog.Logger
}

// NewOrchestrator 创建网关数据面流程编排器。
func NewOrchestrator(svcCtx *svc.ServiceContext) *Orchestrator {
	return &Orchestrator{svc: svcCtx, log: xlog.Component("domain.flow.gateway")}
}

// AccountConfig 解密 Provider 运行所需的账号快照。
func (o *Orchestrator) AccountConfig(row *models.ChannelAccount) (corechannel.AccountConfig, error) {
	if o == nil || o.svc == nil || o.svc.SecretCipher == nil {
		return corechannel.AccountConfig{}, xerr.Internal("渠道凭据加密器未初始化", nil)
	}
	if row == nil {
		return corechannel.AccountConfig{}, xerr.BadRequest("渠道账号不能为空")
	}
	credentials := make(map[string]string, len(row.EncryptedCredentials))
	for key, raw := range row.EncryptedCredentials {
		ciphertext, ok := raw.(string)
		if !ok {
			return corechannel.AccountConfig{}, xerr.Internal("渠道凭据格式无效", nil)
		}
		plain, err := o.svc.SecretCipher.Decrypt(ciphertext)
		if err != nil {
			return corechannel.AccountConfig{}, xerr.Internal("渠道凭据解密失败", err)
		}
		credentials[key] = plain
	}
	settings := make(map[string]any, len(row.Settings))
	for key, value := range row.Settings {
		settings[key] = value
	}
	return corechannel.AccountConfig{
		ID:          row.ID.String(),
		PublicID:    row.PublicID,
		Credentials: credentials,
		Settings:    settings,
	}, nil
}
