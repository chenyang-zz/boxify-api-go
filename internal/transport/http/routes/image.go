/*
 * @Time   : 2026-07-12 20:29:12
 * @Author : chenyang
 * @File   : image.go
 */

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterImageRoutes(api *gin.RouterGroup, image handler.ImageHandler, authMiddleware gin.HandlerFunc) {
	imageGroup := api.Group("image", authMiddleware)

	// @auth(user_id)
	// @description 上传图片
	// @input request.UploadImageRequest
	// @output response.ImageResponse
	imageGroup.POST("/upload", image.UploadImage)

	// @auth(user_id)
	// @description 查询图片列表
	// @input request.ListImagesRequest
	// @output response.PageListResponse[*response.ImageResponse]
	imageGroup.GET("/", image.ListImages)
	imageGroup.GET("/list", image.ListImages)

	// @auth(user_id)
	// @description 检索图片
	// @input request.SearchImageRequest
	// @output response.ListResponse[*response.SearchImageResponse]
	imageGroup.POST("/search", image.SearchImages)

	// @auth(user_id)
	// @description 移动图片到指定知识库
	// @input request.MoveImageRequest
	// @output response.ImageResponse
	imageGroup.POST("/:image_id/move", image.MoveImage)

	// @auth(user_id)
	// @description 获取图片详情
	// @input request.UriImageIDRequest
	// @output response.ImageResponse
	imageGroup.GET("/:image_id", image.GetImage)

	// @auth(user_id)
	// @description 删除图片
	// @input request.UriImageIDRequest
	imageGroup.DELETE("/:image_id", image.DeleteImage)
	imageGroup.POST("/:image_id/delete", image.DeleteImage)
}
