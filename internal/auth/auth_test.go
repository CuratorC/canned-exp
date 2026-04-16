package auth

import (
	"testing"
	"time"
)

// --- TOTP 校验单元测试 ---

func TestAuth_ValidateTOTP(t *testing.T) {
	secret := generateTestSecret(t)
	auth := NewAuth(Config{
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
	auth := NewAuth(Config{
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
		expiredAuth := NewAuth(Config{
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
	auth := NewAuth(Config{
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
	auth := NewAuth(Config{
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
