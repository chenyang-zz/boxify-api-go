/**
 * @Time   : 2026/6/28 15:33
 * @Author : chenyangzhao542@gmail.com
 * @File   : agentconfig.go
 **/

package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterAgentConfigRoutes(api *gin.RouterGroup, agentConfig handler.AgentConfigHandler, authMiddleware gin.HandlerFunc) {
	agentConfigRoutes := api.Group("/agent-config", authMiddleware)
	// @auth(user_id)
	// @description 查询智能体配置
	// @output response.AgentConfigResponse
	agentConfigRoutes.GET("", agentConfig.GetAgentConfig)
	// @auth(user_id)
	// @description 更新智能体配置
	// @input request.UpdateAgentConfigRequest
	// @output response.AgentConfigResponse
	agentConfigRoutes.PUT("", agentConfig.UpdateAgentConfig)
	agentConfigRoutes.POST("/update", agentConfig.UpdateAgentConfig)
	// @auth(user_id)
	// @description 优化提示词
	// 调用默认对话模型，按元提示词把用户的 system_prompt 改写得更专业
	// @input request.OptimizePromptRequest
	// @output response.OptimizePromptResponse
	agentConfigRoutes.POST("/optimize-prompt", agentConfig.OptimizePrompt)
}
