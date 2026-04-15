package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/pquerna/otp/totp"
)

// AuthConfig 认证配置
type AuthConfig struct {
	TOTPSecret string
	APIKey     string
	SessionTTL time.Duration
}

// Session 会话
type Session struct {
	Token     string
	Service   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Auth 认证管理器
type Auth struct {
	config   AuthConfig
	sessions sync.Map // token → *Session
	services sync.Map // service → current active token
}

// NewAuth 创建认证管理器
func NewAuth(config AuthConfig) *Auth {
	return &Auth{config: config}
}

// ValidateTOTP 校验 TOTP 码
func (a *Auth) ValidateTOTP(code string) bool {
	if code == "" || len(code) != 6 {
		return false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return false
		}
	}
	if a.config.TOTPSecret == "" {
		return false
	}
	return totp.Validate(code, a.config.TOTPSecret)
}

// CreateSession 创建会话，自动吊销同服务的旧 session
func (a *Auth) CreateSession(service string) (string, time.Time) {
	// 吊销该服务之前的 session
	a.RevokeService(service)

	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	now := time.Now()
	expiresAt := now.Add(a.config.SessionTTL)

	a.sessions.Store(token, &Session{
		Token:     token,
		Service:   service,
		CreatedAt: now,
		ExpiresAt: expiresAt,
	})
	a.services.Store(service, token)

	return token, expiresAt
}

// ValidateToken 校验 session token
func (a *Auth) ValidateToken(token string) bool {
	if token == "" {
		return false
	}
	val, ok := a.sessions.Load(token)
	if !ok {
		return false
	}
	session := val.(*Session)
	return time.Now().Before(session.ExpiresAt)
}

// ValidateAPIKey 校验 API Key
func (a *Auth) ValidateAPIKey(key string) bool {
	if key == "" || a.config.APIKey == "" {
		return false
	}
	return key == a.config.APIKey
}

// RevokeService 吊销指定服务的当前 session
func (a *Auth) RevokeService(service string) {
	if oldToken, ok := a.services.LoadAndDelete(service); ok {
		a.sessions.Delete(oldToken)
	}
}

// isLoopback 判断请求是否来自本地回环地址
func isLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// LoginHandler 返回 POST /api/auth/login 的 http.Handler
func (a *Auth) LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 未配置 TOTP
		if a.config.TOTPSecret == "" {
			http.Error(w, "TOTP not configured", http.StatusForbidden)
			return
		}

		var req struct {
			TOTPCode string `json:"totp_code"`
			Service  string `json:"service"`
		}
		if err := sonic.ConfigFastest.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Service == "" {
			http.Error(w, "service is required", http.StatusBadRequest)
			return
		}

		if !a.ValidateTOTP(req.TOTPCode) {
			http.Error(w, "invalid totp code", http.StatusUnauthorized)
			return
		}

		token, expiresAt := a.CreateSession(req.Service)

		resp := map[string]interface{}{
			"token":      token,
			"service":    req.Service,
			"expires_at": expiresAt.Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := sonic.Marshal(resp)
		w.Write(data)
	})
}

// RevokeHandler 返回 POST /api/auth/revoke 的 http.Handler
func (a *Auth) RevokeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 校验调用者 token
		token := extractBearerToken(r)
		if !a.ValidateToken(token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req struct {
			Service string `json:"service"`
		}
		if err := sonic.ConfigFastest.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Service == "" {
			http.Error(w, "service is required", http.StatusBadRequest)
			return
		}

		a.RevokeService(req.Service)

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
}

// Middleware 返回认证中间件
func (a *Auth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 免认证路径
		if r.URL.Path == "/api/auth/login" || r.URL.Path == "/api/auth/revoke" {
			next.ServeHTTP(w, r)
			return
		}

		// 本地回环地址免认证（临时禁用，验证 token 校验）
		// if isLoopback(r.RemoteAddr) {
		// 	next.ServeHTTP(w, r)
		// 	return
		// }

		// 读取 Bearer token
		token := extractBearerToken(r)
		if token == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// 先尝试 session token（全权限）
		if a.ValidateToken(token) {
			next.ServeHTTP(w, r)
			return
		}

		// 再尝试 API Key（只读，仅 /api/search）
		if a.ValidateAPIKey(token) {
			if r.URL.Path == "/api/search" {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "api key only allows /api/search", http.StatusForbidden)
			return
		}

		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}

// extractBearerToken 从 Authorization 头提取 Bearer token
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
