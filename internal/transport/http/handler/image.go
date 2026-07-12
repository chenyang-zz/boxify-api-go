package handler

import (
	imagelogic "github.com/boxify/api-go/internal/logic/image"
	"github.com/boxify/api-go/internal/svc"
	"github.com/boxify/api-go/internal/transport/http/request"
	"github.com/boxify/api-go/internal/transport/http/response"
	"github.com/boxify/api-go/internal/util"
	"github.com/boxify/api-go/internal/xerr"
	"github.com/gin-gonic/gin"
)

type ImageHandler struct {
	svc *svc.ServiceContext
}

func NewImageHandler(svcCtx *svc.ServiceContext) ImageHandler {
	return ImageHandler{svc: svcCtx}
}

func (h ImageHandler) UploadImage(c *gin.Context) {
	var body request.UploadImageRequest
	if err := c.ShouldBind(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := imagelogic.NewUploadImageLogic(c.Request.Context(), h.svc).UploadImage(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h ImageHandler) ListImages(c *gin.Context) {
	var query request.ListImagesRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := imagelogic.NewListImagesLogic(c.Request.Context(), h.svc).ListImages(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h ImageHandler) SearchImages(c *gin.Context) {
	var body request.SearchImageRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := imagelogic.NewSearchImagesLogic(c.Request.Context(), h.svc).SearchImages(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h ImageHandler) MoveImage(c *gin.Context) {
	var body request.MoveImageRequest
	if err := c.ShouldBindUri(&body.UriImageIDRequest); err != nil {
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
	out, err := imagelogic.NewMoveImageLogic(c.Request.Context(), h.svc).MoveImage(userID, &body)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h ImageHandler) GetImage(c *gin.Context) {
	var query request.UriImageIDRequest
	if err := c.ShouldBindUri(&query); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	out, err := imagelogic.NewGetImageLogic(c.Request.Context(), h.svc).GetImage(userID, &query)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, out)
}

func (h ImageHandler) DeleteImage(c *gin.Context) {
	var body request.UriImageIDRequest
	if err := c.ShouldBindUri(&body); err != nil {
		response.FromError(c, xerr.Validation(err))
		return
	}
	userID, err := util.UserIDFromContext(c.Request.Context())
	if err != nil {
		response.FromError(c, err)
		return
	}
	if err := imagelogic.NewDeleteImageLogic(c.Request.Context(), h.svc).DeleteImage(userID, &body); err != nil {
		response.FromError(c, err)
		return
	}
	response.OK(c, nil)
}
