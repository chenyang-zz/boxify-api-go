package gateway

import (
	"testing"

	"github.com/boxify/api-go/internal/infrastructure/security"
	"github.com/boxify/api-go/internal/models"
	"github.com/boxify/api-go/internal/svc"
	"github.com/google/uuid"
)

// TestAccountConfigDecryptsProviderSnapshot 验证流程编排器会解密凭据并复制 Provider 运行设置。
func TestAccountConfigDecryptsProviderSnapshot(t *testing.T) {
	cipher, err := security.NewSecretCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	encrypted, err := cipher.Encrypt("provider-secret")
	if err != nil {
		t.Fatal(err)
	}
	accountID := uuid.New()
	settings := models.JSONMap{"callback_url": "https://example.com/reply"}
	flow := NewOrchestrator(&svc.ServiceContext{SecretCipher: cipher})

	got, err := flow.AccountConfig(&models.ChannelAccount{
		ID: accountID, PublicID: "public-id",
		EncryptedCredentials: models.JSONMap{"token": encrypted}, Settings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != accountID.String() || got.PublicID != "public-id" || got.Credentials["token"] != "provider-secret" {
		t.Fatalf("unexpected account config: %#v", got)
	}
	got.Settings["callback_url"] = "changed"
	if settings["callback_url"] != "https://example.com/reply" {
		t.Fatal("account config settings must not alias the persisted model")
	}
}
