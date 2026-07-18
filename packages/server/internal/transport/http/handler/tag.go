package handler

import (
	taglogic "github.com/boxify/api-go/internal/logic/tag"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type TagHandler struct {
	svc *svc.ServiceContext
}

func NewTagHandler(svcCtx *svc.ServiceContext) TagHandler {
	return TagHandler{svc: svcCtx}
}

func (h TagHandler) ListTags(c *gin.Context) {
	var query request.ListTagsRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := taglogic.NewListTagsLogic(c.Request.Context(), h.svc).ListTags(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h TagHandler) UpdateTag(c *gin.Context) {
	var body request.TagUpdateRequest
	if err := c.ShouldBindUri(&body.UriTagServerIDRequest); err != nil {
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
	out, err := taglogic.NewUpdateTagLogic(c.Request.Context(), h.svc).UpdateTag(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h TagHandler) MergeTag(c *gin.Context) {
	var body request.TagMergeRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := taglogic.NewMergeTagLogic(c.Request.Context(), h.svc).MergeTag(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h TagHandler) DeleteTag(c *gin.Context) {
	var body request.UriTagServerIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := taglogic.NewDeleteTagLogic(c.Request.Context(), h.svc).DeleteTag(userID, &body); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
