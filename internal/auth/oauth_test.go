package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

// --- OAuth Client Registration 测试 ---

func TestAuth_RegisterClient(t *testing.T) {
	authSvc := NewAuth(Config{
		TOTPSecret: "dummy",
		SessionTTL: time.Hour,
	})

	t.Run("注册客户端返回 client_id 和 client_secret", func(t *testing.T) {
		client := authSvc.RegisterClient("Claude Code", []string{"http://localhost:8080/callback"})
		if client.ClientID == "" {
			t.Error("client_id 不应为空")
		}
		if client.ClientSecret == "" {
			t.Error("client_secret 不应为空")
		}
		if client.ClientName != "Claude Code" {
			t.Errorf("client_name = %s, want Claude Code", client.ClientName)
		}
		if len(client.RedirectURIs) != 1 || client.RedirectURIs[0] != "http://localhost:8080/callback" {
			t.Errorf("redirect_uris = %v, want [http://localhost:8080/callback]", client.RedirectURIs)
		}
	})

	t.Run("同一个 client_name 多次注册返回不同 client_id", func(t *testing.T) {
		c1 := authSvc.RegisterClient("test-app", []string{"http://localhost/a"})
		c2 := authSvc.RegisterClient("test-app", []string{"http://localhost/a"})
		if c1.ClientID == c2.ClientID {
			t.Error("重复注册应返回不同的 client_id")
		}
	})
}

func TestAuth_GetClient(t *testing.T) {
	authSvc := NewAuth(Config{
		TOTPSecret: "dummy",
		SessionTTL: time.Hour,
	})

	t.Run("存在的 client_id 返回客户端", func(t *testing.T) {
		client := authSvc.RegisterClient("test", []string{"http://localhost/cb"})
		got := authSvc.GetClient(client.ClientID)
		if got == nil {
			t.Fatal("应找到已注册的客户端")
		}
		if got.ClientID != client.ClientID {
			t.Errorf("client_id = %s, want %s", got.ClientID, client.ClientID)
		}
	})

	t.Run("不存在的 client_id 返回 nil", func(t *testing.T) {
		if authSvc.GetClient("nonexistent") != nil {
			t.Error("不存在的 client_id 应返回 nil")
		}
	})
}

func TestAuth_ValidateClient(t *testing.T) {
	authSvc := NewAuth(Config{
		TOTPSecret: "dummy",
		SessionTTL: time.Hour,
	})

	t.Run("正确的 client_id + client_secret 通过", func(t *testing.T) {
		client := authSvc.RegisterClient("test", []string{"http://localhost/cb"})
		if !authSvc.ValidateClient(client.ClientID, client.ClientSecret) {
			t.Error("正确的凭据应通过校验")
		}
	})

	t.Run("错误的 client_secret 失败", func(t *testing.T) {
		client := authSvc.RegisterClient("test", []string{"http://localhost/cb"})
		if authSvc.ValidateClient(client.ClientID, "wrong-secret") {
			t.Error("错误的 client_secret 不应通过")
		}
	})

	t.Run("不存在的 client_id 失败", func(t *testing.T) {
		if authSvc.ValidateClient("nonexistent", "any") {
			t.Error("不存在的 client_id 不应通过")
		}
	})
}

// --- Authorization Code 测试 ---

func TestAuth_AuthorizationCode(t *testing.T) {
	authSvc := NewAuth(Config{
		TOTPSecret: "dummy",
		SessionTTL: time.Hour,
	})

	t.Run("创建并交换授权码成功", func(t *testing.T) {
		client := authSvc.RegisterClient("test", []string{"http://localhost/cb"})
		code := authSvc.CreateAuthorizationCode(client.ClientID, "http://localhost/cb", "", "oauth:svc")

		cid, redirectURI, _, service, err := authSvc.ExchangeAuthorizationCode(code, client.ClientID, "")
		if err != nil {
			t.Fatalf("交换授权码不应报错: %v", err)
		}
		if cid != client.ClientID {
			t.Errorf("client_id = %s, want %s", cid, client.ClientID)
		}
		if redirectURI != "http://localhost/cb" {
			t.Errorf("redirect_uri = %s, want http://localhost/cb", redirectURI)
		}
		if service != "oauth:svc" {
			t.Errorf("service = %s, want oauth:svc", service)
		}
	})

	t.Run("授权码只能使用一次（重放失败）", func(t *testing.T) {
		client := authSvc.RegisterClient("replay-test", []string{"http://localhost/cb"})
		code := authSvc.CreateAuthorizationCode(client.ClientID, "http://localhost/cb", "", "svc")

		_, _, _, _, err := authSvc.ExchangeAuthorizationCode(code, client.ClientID, "")
		if err != nil {
			t.Fatalf("首次交换不应报错: %v", err)
		}

		_, _, _, _, err = authSvc.ExchangeAuthorizationCode(code, client.ClientID, "")
		if err != ErrInvalidGrant {
			t.Errorf("重复使用应返回 ErrInvalidGrant, got %v", err)
		}
	})

	t.Run("不存在的授权码返回 ErrInvalidGrant", func(t *testing.T) {
		_, _, _, _, err := authSvc.ExchangeAuthorizationCode("nonexistent", "any", "")
		if err != ErrInvalidGrant {
			t.Errorf("不存在的 code 应返回 ErrInvalidGrant, got %v", err)
		}
	})

	t.Run("client_id 不匹配返回 ErrInvalidClient", func(t *testing.T) {
		client := authSvc.RegisterClient("mismatch-test", []string{"http://localhost/cb"})
		code := authSvc.CreateAuthorizationCode(client.ClientID, "http://localhost/cb", "", "svc")

		_, _, _, _, err := authSvc.ExchangeAuthorizationCode(code, "wrong-client-id", "")
		if err != ErrInvalidClient {
			t.Errorf("client_id 不匹配应返回 ErrInvalidClient, got %v", err)
		}
	})

	t.Run("PKCE 校验：正确的 code_verifier 通过", func(t *testing.T) {
		client := authSvc.RegisterClient("pkce-test", []string{"http://localhost/cb"})

		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		h := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

		code := authSvc.CreateAuthorizationCode(client.ClientID, "http://localhost/cb", codeChallenge, "svc")

		_, _, _, _, err := authSvc.ExchangeAuthorizationCode(code, client.ClientID, codeVerifier)
		if err != nil {
			t.Fatalf("PKCE 校验应通过: %v", err)
		}
	})

	t.Run("PKCE 校验：错误的 code_verifier 失败", func(t *testing.T) {
		client := authSvc.RegisterClient("pkce-fail", []string{"http://localhost/cb"})

		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		h := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(h[:])

		code := authSvc.CreateAuthorizationCode(client.ClientID, "http://localhost/cb", codeChallenge, "svc")

		_, _, _, _, err := authSvc.ExchangeAuthorizationCode(code, client.ClientID, "wrong-verifier")
		if err != ErrInvalidGrant {
			t.Errorf("错误的 code_verifier 应返回 ErrInvalidGrant, got %v", err)
		}
	})

	t.Run("过期的授权码返回 ErrInvalidGrant", func(t *testing.T) {
		// 手动创建一个已过期的授权码
		code := generateRandomHex(32)
		authSvc.authCodes.Store(code, &AuthorizationCode{
			Code:       code,
			ClientID:   "any",
			ExpiresAt:  time.Now().Add(-1 * time.Second),
		})

		_, _, _, _, err := authSvc.ExchangeAuthorizationCode(code, "any", "")
		if err != ErrInvalidGrant {
			t.Errorf("过期的 code 应返回 ErrInvalidGrant, got %v", err)
		}
	})
}

// --- PKCE challenge 计算 ---

func TestComputePKCEChallenge(t *testing.T) {
	t.Run("RFC 7636 附录 B 示例", func(t *testing.T) {
		codeVerifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
		expected := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
		got := computePKCEChallenge(codeVerifier)
		if got != expected {
			t.Errorf("PKCE challenge = %s, want %s", got, expected)
		}
	})
}
