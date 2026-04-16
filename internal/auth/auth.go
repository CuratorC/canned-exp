package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
)

// Config 认证配置
type Config struct {
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
	config   Config
	sessions sync.Map // token → *Session
	services sync.Map // service → current active token
}

// NewAuth 创建认证管理器
func NewAuth(config Config) *Auth {
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

// TOTPConfigured 返回是否已配置 TOTP 密钥
func (a *Auth) TOTPConfigured() bool {
	return a.config.TOTPSecret != ""
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
