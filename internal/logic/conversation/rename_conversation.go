package conversation

import (
	"context"
	"github.com/boxify/api-go/internal/observability/xlog"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/google/uuid"
	"log/slog"
)

// RenameConversationLogic contains the renameConversation use case.
type RenameConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	log    *slog.Logger
}

// NewRenameConversationLogic creates a RenameConversationLogic.
func NewRenameConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RenameConversationLogic {
	return &RenameConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		log:    xlog.Component("logic.conversation.renameconversation"),
	}
}

// RenameConversation 重命名会话
func (l *RenameConversationLogic) RenameConversation(userID uuid.UUID, input *request.RenameConversationRequest) (*response.ConversationResponse, error) {
	_ = l
	return nil, nil
}
