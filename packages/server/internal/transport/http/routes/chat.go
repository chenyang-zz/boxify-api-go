package routes

import (
	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(api *gin.RouterGroup, chat handler.ChatHandler, authMiddleware gin.HandlerFunc) {
	chatRoutes := api.Group("/chat", authMiddleware)
	// @auth(user_id)
	// @sse
	// @event response.SSEEvent
	// @description 流式聊天
	// @input request.ChatStreamRequest
	chatRoutes.POST("/stream", chat.ChatStream)
}
