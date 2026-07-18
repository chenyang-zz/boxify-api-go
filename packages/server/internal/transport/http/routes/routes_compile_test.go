package routes

import (
	"testing"

	"github.com/boxify/api-go/internal/transport/http/handler"
	"github.com/gin-gonic/gin"
)

func TestRouteRegistrationHelpersAreDefined(t *testing.T) {
	t.Helper()

	var _ func(*gin.RouterGroup, handler.HealthHandler) = RegisterHealthRoutes
	var _ func(*gin.RouterGroup, handler.AuthHandler, gin.HandlerFunc) = RegisterAuthRoutes
	var _ func(*gin.RouterGroup, handler.ChatHandler, gin.HandlerFunc) = RegisterChatRoutes
	var _ func(*gin.RouterGroup, handler.ModelConfigHandler, gin.HandlerFunc) = RegisterModelConfigRoutes
	var _ func(*gin.RouterGroup, handler.MCPServerHandler, gin.HandlerFunc) = RegisterMCPServerRoutes
	var _ func(*gin.RouterGroup) = RegisterDebugRoutes
}

func TestRegisterMCPServerRoutesRegistersStandardPatchPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	RegisterMCPServerRoutes(api, handler.MCPServerHandler{}, func(c *gin.Context) {})

	for _, route := range router.Routes() {
		if route.Method == "PATCH" && route.Path == "/api/mcp/:mcp_id" {
			return
		}
	}
	t.Fatalf("PATCH /api/mcp/:mcp_id route was not registered; routes=%+v", router.Routes())
}

func TestRegisterMCPServerRoutesRegistersTogglePathOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api")

	RegisterMCPServerRoutes(api, handler.MCPServerHandler{}, func(c *gin.Context) {})

	seen := map[string]bool{}
	for _, route := range router.Routes() {
		if route.Method == "POST" {
			seen[route.Path] = true
		}
	}
	if !seen["/api/mcp/:mcp_id/toggle"] {
		t.Fatalf("POST /api/mcp/:mcp_id/toggle route was not registered; routes=%+v", router.Routes())
	}
	if seen["/api/mcp/:mcp_id/troggle"] {
		t.Fatalf("POST /api/mcp/:mcp_id/troggle route should not be registered; routes=%+v", router.Routes())
	}
}
