package handler

import (
	chatlogic "github.com/boxify/api-go/internal/logic/chat"
	"github.com/boxify/api-go/internal/mapper"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	svc *svc.ServiceContext
}

func NewChatHandler(svcCtx *svc.ServiceContext) ChatHandler {
	return ChatHandler{svc: svcCtx}
}

func (h ChatHandler) ChatStream(c *gin.Context) {
	var body request.ChatStreamRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	events, err := chatlogic.NewChatStreamLogic(c.Request.Context(), h.svc).ChatStream(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.StreamEvents(c, mapper.EventStreamToResponse(c.Request.Context(), events))
}
