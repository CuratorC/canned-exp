package controller

import (
	"net/http"
	"time"

	"canned-exp/internal/auth"
	"canned-exp/internal/http/response"

	"github.com/gin-gonic/gin"
)

// loginRequest 登录请求
type loginRequest struct {
	TOTPCode string `json:"totp_code" binding:"required,len=6"`
	Service  string `json:"service" binding:"required"`
}

// revokeRequest 吊销请求
type revokeRequest struct {
	Service string `json:"service" binding:"required"`
}

// LoginHandler 处理 POST /api/auth/login
func LoginHandler(authSvc *auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !authSvc.TOTPConfigured() {
			response.Abort403(c, "TOTP not configured")
			return
		}

		var req loginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err)
			return
		}

		if !authSvc.ValidateTOTP(req.TOTPCode) {
			response.Unauthorized(c, "invalid totp code")
			return
		}

		token, expiresAt := authSvc.CreateSession(req.Service)
		response.Data(c, gin.H{
			"token":      token,
			"service":    req.Service,
			"expires_at": expiresAt.Format(time.RFC3339),
		})
	}
}

// RevokeHandler 处理 POST /api/auth/revoke
func RevokeHandler(authSvc *auth.Auth) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req revokeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err)
			return
		}

		authSvc.RevokeService(req.Service)
		response.Success(c)
	}
}

// GuideHandler 处理 GET /api/auth/guide
// 免认证接口，返回硬编码的认证指南
func GuideHandler() gin.HandlerFunc {
	const guideJSON = `{"title":"MCP Server 认证指南","content":"## 认证方式\n\n本 MCP Server 通过 TOTP 认证保护。Agent 在连接 MCP 之前需先获取 session token。\n\n## 续期步骤\n1. 向用户询问当前 TOTP 验证码（6 位数字，来自 Google Authenticator 等验证器 App）\n2. 调用登录接口获取 session token：\n   curl -s -X POST http://localhost:3100/api/auth/login -H 'Content-Type: application/json' -d '{\"totp_code\":\"<6位码>\",\"service\":\"<主机名>:<工具名>\"}'\n3. 将返回的 token 写入 MCP 配置的 headers.Authorization 字段\n4. 重连 MCP 服务\n\n## Service 参数规则\n格式：系统主机名:工具名（如 myhost:Claude Code）。同一 service 仅保留最新 session。\n\n## 认证层级\n- 本地回环地址(127.0.0.1)：免认证\n- Session Token（通过 TOTP 登录获取）：全权限\n\n## Token 有效期\n默认 24 小时，过期后需重新登录。"}`
	return func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(guideJSON))
	}
}
