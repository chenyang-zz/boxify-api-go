/**
 * @Time   : 2026/7/4 20:51
 * @Author : chenyangzhao542@gmail.com
 * @File   : tag.go
 **/

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterTagRoutes(api *gin.RouterGroup, tag handler.TagHandler, authMiddleware gin.HandlerFunc) {
	tagRoutes := api.Group("/tag", authMiddleware)

	// @auth(user_id)
	// @description 查询标签列表
	// @input request.ListTagsRequest
	// @output response.PageListResponse[*response.TagResponse]
	tagRoutes.GET("", tag.ListTags)
	tagRoutes.GET("/list", tag.ListTags)

	// @auth(user_id)
	// @description 更新标签
	// @input request.TagUpdateRequest
	// @output response.TagResponse
	tagRoutes.PATCH("/:tag_id", tag.UpdateTag)
	tagRoutes.POST("/:tag_id/update", tag.UpdateTag)

	// @auth(user_id)
	// @description 合并标签
	// @input request.TagMergeRequest
	// @output response.TagResponse
	tagRoutes.POST("/merge", tag.MergeTag)

	// @auth(user_id)
	// @description 删除标签
	// @input request.UriTagServerIDRequest
	tagRoutes.DELETE("/:tag_id", tag.DeleteTag)
	tagRoutes.POST("/:tag_id/delete", tag.DeleteTag)
}
