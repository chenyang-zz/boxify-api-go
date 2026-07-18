package handler

import (
	mcpserverlogic "github.com/boxify/api-go/internal/logic/mcpserver"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type MCPServerHandler struct {
	svc *svc.ServiceContext
}

func NewMCPServerHandler(svcCtx *svc.ServiceContext) MCPServerHandler {
	return MCPServerHandler{svc: svcCtx}
}

func (h MCPServerHandler) GetMCPServerList(c *gin.Context) {
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := mcpserverlogic.NewGetMCPServerListLogic(c.Request.Context(), h.svc).GetMCPServerList(userID)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h MCPServerHandler) CreateMCPServer(c *gin.Context) {
	var body request.CreateMCPServerRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := mcpserverlogic.NewCreateMCPServerLogic(c.Request.Context(), h.svc).CreateMCPServer(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h MCPServerHandler) UpdateMCPServer(c *gin.Context) {
	var body request.UpdateMCPServerRequest
	if err := c.ShouldBindUri(&body.UriMCPServerIDRequest); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := mcpserverlogic.NewUpdateMCPServerLogic(c.Request.Context(), h.svc).UpdateMCPServer(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h MCPServerHandler) DeleteMCPServer(c *gin.Context) {
	var body request.UriMCPServerIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := mcpserverlogic.NewDeleteMCPServerLogic(c.Request.Context(), h.svc).DeleteMCPServer(userID, &body); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h MCPServerHandler) SyncMCPServer(c *gin.Context) {
	var body request.UriMCPServerIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := mcpserverlogic.NewSyncMCPServerLogic(c.Request.Context(), h.svc).SyncMCPServer(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h MCPServerHandler) ToggleMCPServer(c *gin.Context) {
	var body request.ToggleMCPServerRequest
	if err := c.ShouldBindUri(&body.UriMCPServerIDRequest); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := mcpserverlogic.NewTroggleMCPServerLogic(c.Request.Context(), h.svc).ToggleMCPServer(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}
