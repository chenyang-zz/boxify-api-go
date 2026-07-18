/**
 * @Time   : 2026/6/29 16:43
 * @Author : chenyangzhao542@gmail.com
 * @File   : mcp.go
 **/

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterMCPServerRoutes(api *gin.RouterGroup, mcp handler.MCPServerHandler, authMiddleware gin.HandlerFunc) {
	mcpRoutes := api.Group("/mcp", authMiddleware)

	// @auth(user_id)
	// @description 查询mcp服务列表
	// @output response.ListResponse[*response.MCPServerResponse]
	mcpRoutes.GET("/", mcp.GetMCPServerList)
	mcpRoutes.GET("/list", mcp.GetMCPServerList)

	// @auth(user_id)
	// @description 新建mcp服务
	// @input request.CreateMCPServerRequest
	// @output response.MCPServerResponse
	mcpRoutes.POST("/", mcp.CreateMCPServer)
	mcpRoutes.POST("/create", mcp.CreateMCPServer)

	// @auth(user_id)
	// @description 更新mcp服务
	// @input request.UpdateMCPServerRequest
	// @output response.MCPServerResponse
	mcpRoutes.PATCH("/:mcp_id", mcp.UpdateMCPServer)
	mcpRoutes.POST("/:mcp_id/update", mcp.UpdateMCPServer)

	// @auth(user_id)
	// @description 删除mcp服务
	// @input request.UriMCPServerIDRequest
	mcpRoutes.DELETE("/:mcp_id", mcp.DeleteMCPServer)
	mcpRoutes.POST("/:mcp_id/delete", mcp.DeleteMCPServer)

	// @auth(user_id)
	// @description 同步mcp服务
	// @input request.UriMCPServerIDRequest
	// @output response.MCPServerResponse
	mcpRoutes.POST("/:mcp_id/sync", mcp.SyncMCPServer)

	// @auth(user_id)
	// @description 切换mcp状态
	// @input request.ToggleMCPServerRequest
	// @output response.MCPServerResponse
	mcpRoutes.POST("/:mcp_id/toggle", mcp.ToggleMCPServer)

}
