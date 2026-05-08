package route

import (
	"canned-exp/internal/app"
	expctrl "canned-exp/internal/mcp/controller"
	mcpgw "canned-exp/internal/mcp"
	httpctrl "canned-exp/internal/http/controller"
	middlewares "canned-exp/internal/http/middleware"
	"canned-exp/internal/proxy"
	"canned-exp/internal/renderer"

	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes 注册所有 API 路由（Web + MCP + Auth）
func RegisterAPIRoutes(r *gin.Engine, application *app.App) {

	// 根路径健康检查，避免 HEAD / 产生 404 WARN 日志
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("health-check", func(c *gin.Context) {
		c.JSON(200, gin.H{})
	})

	r.GET("health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// OAuth controller
	oauthCtrl := httpctrl.NewOAuthController(application.Auth, application.BaseURL)

	// 免认证路由
	r.GET("/api/auth/guide", httpctrl.GuideHandler())
	r.POST("/api/auth/login", httpctrl.LoginHandler(application.Auth))

	// OAuth 2.0 端点（免认证）
	r.GET("/.well-known/oauth-protected-resource", oauthCtrl.ProtectedResourceMetadata)
	r.GET("/.well-known/oauth-authorization-server", oauthCtrl.AuthorizationServerMetadata)
	r.POST("/oauth/register", oauthCtrl.Register)
	r.GET("/oauth/authorize", oauthCtrl.AuthorizeGet)
	r.POST("/oauth/authorize", oauthCtrl.AuthorizePost)
	r.POST("/oauth/token", oauthCtrl.Token)

	// 需认证路由
	authGroup := r.Group("", middlewares.AuthMiddleware(application.Auth))
	authGroup.POST("/api/auth/revoke", httpctrl.RevokeHandler(application.Auth))
	authGroup.POST("/api/search", httpctrl.SearchHandler(application.ExperienceService))

	// MCP SSE（SDK http.Handler 用 gin.WrapH 包装）
	sseServer := mcpgw.NewSSEServer("canned-exp", "1.0.0", expctrl.NewServer(application.ExperienceService, application.AgentService, application.PersonalityService, application.PersonalityKeyService, application.MemoryService, renderer.NewRegistry()))
	authGroup.Any("/sse", gin.WrapH(sseServer))
	authGroup.Any("/message", gin.WrapH(sseServer))

	// MCP Streamable HTTP（供 Hermes 等新客户端使用）
	streamableServer := mcpgw.NewStreamableHTTPServer("canned-exp", "1.0.0", expctrl.NewServer(application.ExperienceService, application.AgentService, application.PersonalityService, application.PersonalityKeyService, application.MemoryService, renderer.NewRegistry()))
	authGroup.Any("/mcp", gin.WrapH(streamableServer))

	// Anthropic Messages API 代理（可选，通过 PROXY_ENABLED 开启）
	if application.ProxyEnabled {
		authGroup.POST("/v1/messages", proxy.MessagesHandler(application.ProxyService))
	}

	// v1 Web API 路由组
	v1 := r.Group("/v1")
	v1.Use(middlewares.LimitIP("200-H"))
	{
		_ = v1 // 后续注册 Web API 路由
	}
}
