package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"canned-exp/internal/auth"
	"github.com/gin-gonic/gin"
)

// --- 登录端点测试 ---

func TestLoginHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := generateTestSecret(t)
	authSvc := auth.NewAuth(auth.Config{
		TOTPSecret: secret,
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})

	setupRouter := func(svc *auth.Auth) (*gin.Engine, *httptest.ResponseRecorder) {
		r := gin.New()
		r.POST("/api/auth/login", LoginHandler(svc))
		return r, httptest.NewRecorder()
	}

	t.Run("正确 TOTP 码 + service 登录返回 token", func(t *testing.T) {
		code := generateTestCode(t, secret)
		router, w := setupRouter(authSvc)
		body, _ := json.Marshal(map[string]string{"totp_code": code, "service": "my-service"})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)

		// response.Data 包裹为 {"success": true, "data": {...}}
		data, ok := resp["data"].(map[string]interface{})
		if !ok {
			t.Fatal("响应应包含 data 字段")
		}
		if data["token"] == nil || data["token"] == "" {
			t.Error("响应应包含 token")
		}
		if data["service"] != "my-service" {
			t.Errorf("响应 service = %v, want my-service", data["service"])
		}
		if data["expires_at"] == nil || data["expires_at"] == "" {
			t.Error("响应应包含 expires_at")
		}
	})

	t.Run("缺少 service 参数返回 400", func(t *testing.T) {
		code := generateTestCode(t, secret)
		router, w := setupRouter(authSvc)
		body, _ := json.Marshal(map[string]string{"totp_code": code})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("错误 TOTP 码返回 401", func(t *testing.T) {
		router, w := setupRouter(authSvc)
		body, _ := json.Marshal(map[string]string{"totp_code": "000000", "service": "svc"})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})

	t.Run("未配置 TOTP secret 返回 403", func(t *testing.T) {
		noTotpAuthSvc := auth.NewAuth(auth.Config{
			TOTPSecret: "",
			APIKey:     "test-api-key",
			SessionTTL: time.Hour,
		})
		router, w := setupRouter(noTotpAuthSvc)

		body, _ := json.Marshal(map[string]string{"totp_code": "123456", "service": "svc"})
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", w.Code)
		}
	})
}

// --- Revoke 端点测试 ---

func TestRevokeHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	authSvc := auth.NewAuth(auth.Config{
		TOTPSecret: "dummy",
		APIKey:     "test-api-key",
		SessionTTL: time.Hour,
	})

	t.Run("有效 token 吊销指定服务", func(t *testing.T) {
		targetToken, _ := authSvc.CreateSession("revoke-target")
		callerToken, _ := authSvc.CreateSession("caller-svc")

		r := gin.New()
		r.POST("/api/auth/revoke", RevokeHandler(authSvc))
		w := httptest.NewRecorder()

		body, _ := json.Marshal(map[string]string{"service": "revoke-target"})
		req := httptest.NewRequest("POST", "/api/auth/revoke", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", w.Code)
		}
		if authSvc.ValidateToken(targetToken) {
			t.Error("被吊销服务的 token 应失效")
		}
		if !authSvc.ValidateToken(callerToken) {
			t.Error("调用者的 token 不应受影响")
		}
	})

	t.Run("缺少 service 参数返回 400", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/auth/revoke", RevokeHandler(authSvc))
		w := httptest.NewRecorder()

		req := httptest.NewRequest("POST", "/api/auth/revoke", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}

// --- GuideHandler 测试 ---

func TestGuideHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("免认证返回 200 和认证指南", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/auth/guide", GuideHandler())
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/guide", nil)

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("响应不是有效 JSON: %v", err)
		}
		if resp["title"] == nil || resp["title"] == "" {
			t.Error("响应应包含 title")
		}
		if resp["content"] == nil || resp["content"] == "" {
			t.Error("响应应包含 content")
		}
	})

	t.Run("响应内容包含认证关键词", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/auth/guide", GuideHandler())
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/guide", nil)

		r.ServeHTTP(w, req)

		body := w.Body.String()
		for _, keyword := range []string{"TOTP", "token", "/api/auth/login"} {
			if !strings.Contains(body, keyword) {
				t.Errorf("认证指南应包含关键词 %q", keyword)
			}
		}
	})

	t.Run("响应不包含任何用户经验数据", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/auth/guide", GuideHandler())
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/auth/guide", nil)

		r.ServeHTTP(w, req)

		body := w.Body.String()
		if strings.Contains(body, "experience") || strings.Contains(body, "score") {
			t.Error("认证指南不应包含经验库数据字段")
		}
	})
}
