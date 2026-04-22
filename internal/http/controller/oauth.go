package controller

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"canned-exp/internal/auth"
	"canned-exp/internal/http/response"

	"github.com/gin-gonic/gin"
)

// OAuthController 处理 OAuth 2.0 相关端点
type OAuthController struct {
	authSvc *auth.Auth
	baseURL string
}

// NewOAuthController 创建 OAuth controller
func NewOAuthController(authSvc *auth.Auth, baseURL string) *OAuthController {
	return &OAuthController{authSvc: authSvc, baseURL: baseURL}
}

// --- RFC 9728: Protected Resource Metadata ---

func (ctrl *OAuthController) ProtectedResourceMetadata(c *gin.Context) {
	base := requestBaseURL(c, ctrl.baseURL)
	response.OAuthJSON(c, http.StatusOK, gin.H{
		"authorization_servers":    []string{base},
		"resource":                 base,
		"scopes_supported":         []string{"mcp"},
		"bearer_methods_supported": []string{"header"},
		"resource_documentation":   base + "/api/auth/guide",
	})
}

// --- RFC 8414: Authorization Server Metadata ---

func (ctrl *OAuthController) AuthorizationServerMetadata(c *gin.Context) {
	base := requestBaseURL(c, ctrl.baseURL)
	response.OAuthJSON(c, http.StatusOK, gin.H{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"registration_endpoint":                 base + "/oauth/register",
		"response_types_supported":              []string{"code"},
		"code_challenge_methods_supported":      []string{"S256"},
		"grant_types_supported":                 []string{"authorization_code"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic", "none"},
		"scopes_supported":                      []string{"mcp"},
	})
}

// --- RFC 7591: Dynamic Client Registration ---

type registerRequest struct {
	ClientName              string   `json:"client_name" binding:"required"`
	RedirectURIs            []string `json:"redirect_uris" binding:"required,min=1"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

func (ctrl *OAuthController) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	authMethod := req.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "client_secret_post"
	}

	client := ctrl.authSvc.RegisterClient(req.ClientName, req.RedirectURIs, authMethod)

	resp := gin.H{
		"client_id":                  client.ClientID,
		"client_name":                client.ClientName,
		"redirect_uris":              client.RedirectURIs,
		"token_endpoint_auth_method": client.AuthMethod,
	}
	if client.AuthMethod != "none" {
		resp["client_secret"] = client.ClientSecret
	}
	response.OAuthJSON(c, http.StatusCreated, resp)
}

// --- Authorization Endpoint ---

func (ctrl *OAuthController) AuthorizeGet(c *gin.Context) {
	clientID := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	codeChallenge := c.Query("code_challenge")
	state := c.Query("state")

	if clientID == "" || redirectURI == "" || codeChallenge == "" {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", "missing required parameters: client_id, redirect_uri, code_challenge")
		return
	}

	client := ctrl.authSvc.GetClient(clientID)
	if client == nil {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unknown client_id: %s", clientID))
		return
	}

	if !containsURI(client.RedirectURIs, redirectURI) {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", "redirect_uri not registered")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderAuthorizePage(clientID, redirectURI, codeChallenge, state, "")))
}

func (ctrl *OAuthController) AuthorizePost(c *gin.Context) {
	clientID := c.PostForm("client_id")
	redirectURI := c.PostForm("redirect_uri")
	codeChallenge := c.PostForm("code_challenge")
	state := c.PostForm("state")
	totpCode := c.PostForm("totp_code")

	if clientID == "" || redirectURI == "" || codeChallenge == "" {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", "missing required parameters")
		return
	}

	client := ctrl.authSvc.GetClient(clientID)
	if client == nil {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", "unknown client_id")
		return
	}

	if !containsURI(client.RedirectURIs, redirectURI) {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", "redirect_uri not registered")
		return
	}

	if !ctrl.authSvc.TOTPConfigured() {
		response.OAuthError(c, http.StatusForbidden, "server_error", "TOTP not configured")
		return
	}

	if !ctrl.authSvc.ValidateTOTP(totpCode) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(renderAuthorizePage(clientID, redirectURI, codeChallenge, state, "验证码无效，请重试")))
		return
	}

	service := fmt.Sprintf("oauth:%s:%s", clientID, client.ClientName)
	code := ctrl.authSvc.CreateAuthorizationCode(clientID, redirectURI, codeChallenge, service)

	redirectURL, _ := url.Parse(redirectURI)
	q := redirectURL.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	redirectURL.RawQuery = q.Encode()

	c.Redirect(http.StatusFound, redirectURL.String())
}

// --- Token Endpoint ---

type tokenRequest struct {
	GrantType    string `form:"grant_type" binding:"required"`
	Code         string `form:"code" binding:"required"`
	ClientID     string `form:"client_id"`
	ClientSecret string `form:"client_secret"`
	CodeVerifier string `form:"code_verifier"`
	RedirectURI  string `form:"redirect_uri"`
}

func (ctrl *OAuthController) Token(c *gin.Context) {
	var req tokenRequest
	if err := c.ShouldBind(&req); err != nil {
		response.OAuthError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.GrantType != "authorization_code" {
		response.OAuthError(c, http.StatusBadRequest, "unsupported_grant_type")
		return
	}

	// Fallback 1: 从 Authorization: Basic 头提取 client_id/client_secret
	if req.ClientID == "" {
		if cid, csecret, ok := parseBasicAuth(c); ok {
			req.ClientID = cid
			req.ClientSecret = csecret
		}
	}

	// Fallback 2: "none" 认证方式的公共客户端不发送 client_id，从 auth code 反查
	if req.ClientID == "" {
		req.ClientID = ctrl.authSvc.GetClientIDByAuthCode(req.Code)
	}

	if req.ClientID == "" {
		response.OAuthError(c, http.StatusUnauthorized, "invalid_client")
		return
	}

	if !ctrl.authSvc.ValidateClient(req.ClientID, req.ClientSecret) {
		response.OAuthError(c, http.StatusUnauthorized, "invalid_client")
		return
	}

	_, _, _, service, err := ctrl.authSvc.ExchangeAuthorizationCode(req.Code, req.ClientID, req.CodeVerifier)
	if err != nil {
		status := http.StatusBadRequest
		if err == auth.ErrInvalidClient {
			status = http.StatusUnauthorized
		}
		response.OAuthError(c, status, err.Error())
		return
	}

	token, expiresAt := ctrl.authSvc.CreateSession(service)
	expiresIn := int(time.Until(expiresAt).Seconds())
	if expiresIn < 0 {
		expiresIn = 0
	}

	response.OAuthJSON(c, http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "bearer",
		"expires_in":   expiresIn,
	})
}

// --- helpers ---

// requestBaseURL 从请求中动态推断 baseURL，确保 metadata 中的 URL 与客户端实际连接地址一致
func requestBaseURL(c *gin.Context, fallback string) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	if host := c.Request.Host; host != "" {
		return scheme + "://" + host
	}
	return fallback
}

func containsURI(uris []string, target string) bool {
	for _, u := range uris {
		if u == target {
			return true
		}
	}
	return false
}

// parseBasicAuth 从 Authorization: Basic base64(client_id:client_secret) 头提取凭据
func parseBasicAuth(c *gin.Context) (string, string, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Basic ") {
		return "", "", false
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func renderAuthorizePage(clientID, redirectURI, codeChallenge, state, errMsg string) string {
	errHTML := ""
	if errMsg != "" {
		errHTML = fmt.Sprintf(`<p style="color:red">%s</p>`, errMsg)
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh"><head><meta charset="utf-8"><title>MCP Server 授权</title>
<style>body{font-family:system-ui;max-width:400px;margin:80px auto;padding:20px}
input{width:100%%;padding:12px;margin:8px 0;box-sizing:border-box;font-size:18px;text-align:center;letter-spacing:8px}
button{width:100%%;padding:12px;background:#2563eb;color:#fff;border:none;border-radius:6px;font-size:16px;cursor:pointer}
button:hover{background:#1d4ed8}</style></head>
<body>
<h2>MCP Server 授权</h2>
<p>请输入 TOTP 验证码以授权访问：</p>
%s
<form method="POST" action="/oauth/authorize">
<input type="hidden" name="client_id" value="%s">
<input type="hidden" name="redirect_uri" value="%s">
<input type="hidden" name="code_challenge" value="%s">
<input type="hidden" name="state" value="%s">
<input type="text" name="totp_code" pattern="[0-9]{6}" maxlength="6" placeholder="000000" required autofocus>
<button type="submit">授权</button>
</form></body></html>`, errHTML,
		escapeHTML(clientID),
		escapeHTML(redirectURI),
		escapeHTML(codeChallenge),
		escapeHTML(state),
	)
}

func escapeHTML(s string) string {
	var buf bytes.Buffer
	json.HTMLEscape(&buf, []byte(s))
	return buf.String()
}
