package route

import (
	"canned-exp/internal/app"
	expctrl "canned-exp/internal/mcp/controller"
	mcpgw "canned-exp/internal/mcp"
	httpctrl "canned-exp/internal/http/controller"
	middlewares "canned-exp/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes 注册所有 API 路由（Web + MCP + Auth）
func RegisterAPIRoutes(r *gin.Engine, application *app.App) {

	r.GET("health-check", func(c *gin.Context) {
		c.JSON(200, gin.H{})
	})

	// 免认证路由
	r.GET("/api/auth/guide", httpctrl.GuideHandler())
	r.POST("/api/auth/login", httpctrl.LoginHandler(application.Auth))

	// 需认证路由
	authGroup := r.Group("", middlewares.AuthMiddleware(application.Auth))
	authGroup.POST("/api/auth/revoke", httpctrl.RevokeHandler(application.Auth))
	authGroup.POST("/api/search", httpctrl.SearchHandler(application.ExperienceService))

	// MCP SSE（SDK http.Handler 用 gin.WrapH 包装）
	sseServer := mcpgw.NewSSEServer("canned-exp", "1.0.0", expctrl.NewServer(application.ExperienceService, application.AgentService, application.PersonalityService, application.PersonalityKeyService))
	authGroup.Any("/sse", gin.WrapH(sseServer))
	authGroup.Any("/message", gin.WrapH(sseServer))

	// v1 Web API 路由组
	v1 := r.Group("/v1")
	v1.Use(middlewares.LimitIP("200-H"))
	{
		_ = v1 // 后续注册 Web API 路由
	}
}
