package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// generateTestSecret 生成一个可用于测试的 TOTP 密钥
func generateTestSecret(t *testing.T) string {
	t.Helper()
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "canned-exp-test",
		AccountName: "test@test.com",
	})
	if err != nil {
		t.Fatalf("generate totp key: %v", err)
	}
	return key.Secret()
}

// generateTestCode 根据 secret 生成当前有效的 TOTP 码
func generateTestCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}
	return code
}

// --- TOTP 校验单元测试 ---

func TestAuth_ValidateTOTP(t *testing.T) {
	secret := generateTestSecret(t)
	auth := NewAuth(AuthConfig{
		TOTPSecret: secret,
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})

	t.Run("正确的 6 位码验证通过", func(t *testing.T) {
		code := generateTestCode(t, secret)
		if !auth.ValidateTOTP(code) {
			t.Error("正确的 TOTP 码应验证通过")
		}
	})

	t.Run("错误的 TOTP 码验证失败", func(t *testing.T) {
		if auth.ValidateTOTP("000000") {
			t.Error("错误的 TOTP 码不应验证通过")
		}
	})

	t.Run("空字符串验证失败", func(t *testing.T) {
		if auth.ValidateTOTP("") {
			t.Error("空字符串不应验证通过")
		}
	})

	t.Run("包含字母的输入验证失败", func(t *testing.T) {
		if auth.ValidateTOTP("12ab56") {
			t.Error("包含字母的输入不应验证通过")
		}
	})

	t.Run("位数不足验证失败", func(t *testing.T) {
		if auth.ValidateTOTP("12345") {
			t.Error("5 位码不应验证通过")
		}
	})

	t.Run("位数过多验证失败", func(t *testing.T) {
		if auth.ValidateTOTP("1234567") {
			t.Error("7 位码不应验证通过")
		}
	})
}

// --- Session 单元测试 ---

func TestAuth_Session(t *testing.T) {
	auth := NewAuth(AuthConfig{
		TOTPSecret: "dummy",
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})

	t.Run("创建 session 后 token 有效", func(t *testing.T) {
		token, _ := auth.CreateSession("service-a")
		if token == "" {
			t.Fatal("token 不应为空")
		}
		if !auth.ValidateToken(token) {
			t.Error("刚创建的 session 应有效")
		}
	})

	t.Run("过期 session 无效", func(t *testing.T) {
		expiredAuth := NewAuth(AuthConfig{
			TOTPSecret: "dummy",
			APIKey:     "test-api-key",
			SessionTTL: -1 * time.Second, // 立即过期
		})
		token, _ := expiredAuth.CreateSession("service-a")
		if expiredAuth.ValidateToken(token) {
			t.Error("过期的 session 应无效")
		}
	})

	t.Run("同一服务创建新 session 后旧 token 失效", func(t *testing.T) {
		oldToken, _ := auth.CreateSession("service-same")
		if !auth.ValidateToken(oldToken) {
			t.Fatal("旧 token 初始应有效")
		}

		newToken, _ := auth.CreateSession("service-same")

		if auth.ValidateToken(oldToken) {
			t.Error("同服务创建新 session 后，旧 token 应失效")
		}
		if !auth.ValidateToken(newToken) {
			t.Error("新 token 应有效")
		}
	})

	t.Run("不同服务的 session 并列存在", func(t *testing.T) {
		tokenA, _ := auth.CreateSession("svc-a")
		tokenB, _ := auth.CreateSession("svc-b")

		if !auth.ValidateToken(tokenA) {
			t.Error("svc-a 的 token 应有效")
		}
		if !auth.ValidateToken(tokenB) {
			t.Error("svc-b 的 token 应有效")
		}
	})
}

// --- RevokeService 单元测试 ---

func TestAuth_RevokeService(t *testing.T) {
	auth := NewAuth(AuthConfig{
		TOTPSecret: "dummy",
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})

	t.Run("吊销指定服务的 session", func(t *testing.T) {
		token, _ := auth.CreateSession("target-svc")
		if !auth.ValidateToken(token) {
			t.Fatal("创建后应有效")
		}

		auth.RevokeService("target-svc")

		if auth.ValidateToken(token) {
			t.Error("吊销后 token 应失效")
		}
	})

	t.Run("吊销一个服务不影响另一个", func(t *testing.T) {
		tokenA, _ := auth.CreateSession("svc-x")
		tokenB, _ := auth.CreateSession("svc-y")

		auth.RevokeService("svc-x")

		if auth.ValidateToken(tokenA) {
			t.Error("svc-x 被吊销后其 token 应失效")
		}
		if !auth.ValidateToken(tokenB) {
			t.Error("svc-y 不应受影响")
		}
	})

	t.Run("吊销不存在的服务不报错", func(t *testing.T) {
		// 应该是 no-op，不 panic
		auth.RevokeService("non-existent")
	})
}

// --- API Key 单元测试 ---

func TestAuth_ValidateAPIKey(t *testing.T) {
	auth := NewAuth(AuthConfig{
		TOTPSecret: "dummy",
		APIKey:     "my-secret-key-123",
		SessionTTL: time.Hour,
	})

	t.Run("正确的 key 验证通过", func(t *testing.T) {
		if !auth.ValidateAPIKey("my-secret-key-123") {
			t.Error("正确的 API Key 应验证通过")
		}
	})

	t.Run("错误的 key 验证失败", func(t *testing.T) {
		if auth.ValidateAPIKey("wrong-key") {
			t.Error("错误的 API Key 不应验证通过")
		}
	})

	t.Run("空 key 验证失败", func(t *testing.T) {
		if auth.ValidateAPIKey("") {
			t.Error("空 API Key 不应验证通过")
		}
	})
}

// --- 登录端点测试 ---

func TestLoginHandler(t *testing.T) {
	secret := generateTestSecret(t)
	auth := NewAuth(AuthConfig{
		TOTPSecret: secret,
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})
	handler := auth.LoginHandler()

	t.Run("正确 TOTP 码 + service 登录返回 token", func(t *testing.T) {
		code := generateTestCode(t, secret)
		body, _ := json.Marshal(map[string]string{"totp_code": code, "service": "my-service"})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["token"] == nil || resp["token"] == "" {
			t.Error("响应应包含 token")
		}
		if resp["service"] != "my-service" {
			t.Errorf("响应 service = %v, want my-service", resp["service"])
		}
		if resp["expires_at"] == nil || resp["expires_at"] == "" {
			t.Error("响应应包含 expires_at")
		}
	})

	t.Run("缺少 service 参数返回 400", func(t *testing.T) {
		code := generateTestCode(t, secret)
		body, _ := json.Marshal(map[string]string{"totp_code": code})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("错误 TOTP 码返回 401", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"totp_code": "000000", "service": "svc"})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})

	t.Run("未配置 TOTP secret 返回 403", func(t *testing.T) {
		noTotpAuth := NewAuth(AuthConfig{
			TOTPSecret: "",
			APIKey:     "test-api-key",
			SessionTTL: time.Hour,
		})
		handler := noTotpAuth.LoginHandler()

		body, _ := json.Marshal(map[string]string{"totp_code": "123456", "service": "svc"})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})
}

// --- Revoke 端点测试 ---

func TestRevokeHandler(t *testing.T) {
	auth := NewAuth(AuthConfig{
		TOTPSecret: "dummy",
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})
	handler := auth.RevokeHandler()

	t.Run("有效 token 吊销指定服务", func(t *testing.T) {
		targetToken, _ := auth.CreateSession("revoke-target")
		callerToken, _ := auth.CreateSession("caller-svc")

		body, _ := json.Marshal(map[string]string{"service": "revoke-target"})
		req := httptest.NewRequest("POST", "/api/auth/revoke", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+callerToken)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
		if auth.ValidateToken(targetToken) {
			t.Error("被吊销服务的 token 应失效")
		}
		if !auth.ValidateToken(callerToken) {
			t.Error("调用者的 token 不应受影响")
		}
	})

	t.Run("缺少 service 参数返回 400", func(t *testing.T) {
		callerToken, _ := auth.CreateSession("caller2")

		req := httptest.NewRequest("POST", "/api/auth/revoke", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+callerToken)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}

// --- 中间件测试 ---

// mockNext 创建一个总是返回 200 的 mock handler
func mockNext() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
}

func TestMiddleware(t *testing.T) {
	secret := generateTestSecret(t)
	auth := NewAuth(AuthConfig{
		TOTPSecret: secret,
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})

	t.Run("本地回环地址免认证放行", func(t *testing.T) {
		handler := auth.Middleware(mockNext())
		req := httptest.NewRequest("GET", "/api/search", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("本地回环应免认证，status = %d", w.Code)
		}
	})

	t.Run("有效 session token 放行所有路径", func(t *testing.T) {
		token, _ := auth.CreateSession("test-svc")
		handler := auth.Middleware(mockNext())

		for _, path := range []string{"/api/search", "/sse", "/message"} {
			req := httptest.NewRequest("POST", path, nil)
			req.Header.Set("Authorization", "Bearer "+token)
			req.RemoteAddr = "203.0.113.1:12345" // 非本地
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("session 应放行 %s，status = %d", path, w.Code)
			}
		}
	})

	t.Run("有效 API Key 放行 /api/search", func(t *testing.T) {
		handler := auth.Middleware(mockNext())
		req := httptest.NewRequest("POST", "/api/search", nil)
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("API Key 应放行 /api/search，status = %d", w.Code)
		}
	})

	t.Run("API Key 访问 /sse 返回 403", func(t *testing.T) {
		handler := auth.Middleware(mockNext())
		req := httptest.NewRequest("GET", "/sse", nil)
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("API Key 不应访问 /sse，status = %d, want 403", w.Code)
		}
	})

	t.Run("无认证 + 非本地返回 401", func(t *testing.T) {
		handler := auth.Middleware(mockNext())
		req := httptest.NewRequest("POST", "/api/search", nil)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("无认证非本地应返回 401，status = %d", w.Code)
		}
	})

	t.Run("/api/auth/login 免认证放行", func(t *testing.T) {
		handler := auth.Middleware(mockNext())
		req := httptest.NewRequest("POST", "/api/auth/login", nil)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("/api/auth/login 应免认证，status = %d", w.Code)
		}
	})

	t.Run("/api/auth/revoke 免认证放行（由 handler 自行校验）", func(t *testing.T) {
		handler := auth.Middleware(mockNext())
		req := httptest.NewRequest("POST", "/api/auth/revoke", nil)
		req.RemoteAddr = "203.0.113.1:12345"
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("/api/auth/revoke 应免认证放行，status = %d", w.Code)
		}
	})
}
