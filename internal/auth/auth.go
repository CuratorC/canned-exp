package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
)

var (
	ErrInvalidGrant  = errors.New("invalid_grant")
	ErrInvalidClient = errors.New("invalid_client")
)

// Config 认证配置
type Config struct {
	TOTPSecret string
	SessionTTL time.Duration
}

// Session 会话
type Session struct {
	Token     string
	Service   string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// OAuthClient OAuth 动态注册的客户端 (RFC 7591)
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	RedirectURIs []string
	ClientName   string
	CreatedAt    time.Time
}

// AuthorizationCode OAuth 授权码（一次性，短期有效）
type AuthorizationCode struct {
	Code         string
	ClientID     string
	RedirectURI  string
	CodeChallenge string // PKCE S256
	Service      string
	ExpiresAt    time.Time
}

// Auth 认证管理器
type Auth struct {
	config    Config
	sessions  sync.Map // token → *Session
	services  sync.Map // service → current active token
	clients   sync.Map // client_id → *OAuthClient
	authCodes sync.Map // code → *AuthorizationCode
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

// RevokeService 吊销指定服务的当前 session
func (a *Auth) RevokeService(service string) {
	if oldToken, ok := a.services.LoadAndDelete(service); ok {
		a.sessions.Delete(oldToken)
	}
}

// --- OAuth Dynamic Client Registration (RFC 7591) ---

// RegisterClient 注册新的 OAuth 客户端
func (a *Auth) RegisterClient(clientName string, redirectURIs []string) *OAuthClient {
	clientID := generateRandomHex(16)
	clientSecret := generateRandomHex(32)

	client := &OAuthClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURIs: redirectURIs,
		ClientName:   clientName,
		CreatedAt:    time.Now(),
	}
	a.clients.Store(clientID, client)
	return client
}

// GetClient 获取已注册的 OAuth 客户端
func (a *Auth) GetClient(clientID string) *OAuthClient {
	val, ok := a.clients.Load(clientID)
	if !ok {
		return nil
	}
	return val.(*OAuthClient)
}

// ValidateClient 校验 client_id + client_secret
func (a *Auth) ValidateClient(clientID, clientSecret string) bool {
	client := a.GetClient(clientID)
	if client == nil {
		return false
	}
	return client.ClientSecret == clientSecret
}

// --- OAuth Authorization Code ---

// CreateAuthorizationCode 创建授权码（一次性，5 分钟有效）
func (a *Auth) CreateAuthorizationCode(clientID, redirectURI, codeChallenge, service string) string {
	code := generateRandomHex(32)
	a.authCodes.Store(code, &AuthorizationCode{
		Code:          code,
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		CodeChallenge: codeChallenge,
		Service:       service,
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	})
	return code
}

// ExchangeAuthorizationCode 交换授权码，校验后立即删除（一次性使用）
// 返回 (clientID, redirectURI, codeChallenge, service, error)
func (a *Auth) ExchangeAuthorizationCode(code, clientID, codeVerifier string) (string, string, string, string, error) {
	val, ok := a.authCodes.LoadAndDelete(code)
	if !ok {
		return "", "", "", "", ErrInvalidGrant
	}
	ac := val.(*AuthorizationCode)

	if time.Now().After(ac.ExpiresAt) {
		return "", "", "", "", ErrInvalidGrant
	}
	if ac.ClientID != clientID {
		return "", "", "", "", ErrInvalidClient
	}
	// PKCE 校验：code_challenge = base64url(sha256(code_verifier))
	if ac.CodeChallenge != "" {
		computed := computePKCEChallenge(codeVerifier)
		if computed != ac.CodeChallenge {
			return "", "", "", "", ErrInvalidGrant
		}
	}

	return ac.ClientID, ac.RedirectURI, ac.CodeChallenge, ac.Service, nil
}

// generateRandomHex 生成指定字节数的随机 hex 字符串
func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// computePKCEChallenge 按 S256 方法计算 code_challenge
// code_challenge = base64url(sha256(code_verifier))
func computePKCEChallenge(codeVerifier string) string {
	h := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
