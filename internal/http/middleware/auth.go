package middlewares

import (
	"strings"

	"canned-exp/internal/auth"
	"canned-exp/internal/helper"
	"canned-exp/internal/http/response"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
// 本地回环地址免认证，其余需有效 session token
// 应只应用于需要认证的路由，免认证路由不挂载此中间件
func AuthMiddleware(authSvc *auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 本地回环地址免认证
		if helper.IsLoopback(c.ClientIP()) {
			c.Set("auth_bypassed", true)
			c.Next()
			return
		}

		token := extractBearerToken(c)
		if token == "" {
			response.Unauthorized(c, "未授权")
			c.Abort()
			return
		}

		if !authSvc.ValidateToken(token) {
			response.Unauthorized(c, "未授权")
			c.Abort()
			return
		}

		// 认证信息存入 context，下游 handler 可取用
		c.Set("auth_token", token)
		c.Next()
	}
}

// extractBearerToken 从 Authorization 头提取 Bearer token
func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	if auth == "" {
		return ""
	}
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
