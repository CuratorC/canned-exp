package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"canned-exp/internal/auth"
	"github.com/gin-gonic/gin"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secret := generateTestSecret(t)
	authSvc := auth.NewAuth(auth.Config{
		TOTPSecret: secret,
		SessionTTL: time.Hour,
	})

	// setupRouter 创建带 AuthMiddleware 的 Gin 路由，最终 handler 返回 200
	setupRouter := func(svc *auth.Auth) (*gin.Engine, *httptest.ResponseRecorder) {
		r := gin.New()
		r.Use(AuthMiddleware(svc))
		r.Any("/test", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})
		return r, httptest.NewRecorder()
	}

	t.Run("有效 session token 放行", func(t *testing.T) {
		token, _ := authSvc.CreateSession("test-svc")
		router, w := setupRouter(authSvc)

		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.RemoteAddr = "203.0.113.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("有效 session 应放行，status = %d", w.Code)
		}
	})

	t.Run("本地回环地址免认证放行", func(t *testing.T) {
		router, w := setupRouter(authSvc)
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("本地回环应免认证，status = %d", w.Code)
		}
	})

	t.Run("无 Authorization header 返回 401", func(t *testing.T) {
		router, w := setupRouter(authSvc)
		req := httptest.NewRequest("POST", "/test", nil)
		req.RemoteAddr = "203.0.113.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("无认证应返回 401，status = %d", w.Code)
		}
	})

	t.Run("无效 token 返回 401", func(t *testing.T) {
		router, w := setupRouter(authSvc)
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		req.RemoteAddr = "203.0.113.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("无效 token 应返回 401，status = %d", w.Code)
		}
	})

	t.Run("过期 token 返回 401", func(t *testing.T) {
		expiredAuthSvc := auth.NewAuth(auth.Config{
			TOTPSecret: secret,
			SessionTTL: -1 * time.Second,
		})

		token, _ := expiredAuthSvc.CreateSession("expired-svc")
		router, w := setupRouter(expiredAuthSvc)

		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.RemoteAddr = "203.0.113.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("过期 token 应返回 401，status = %d", w.Code)
		}
	})

	t.Run("401 响应包含 WWW-Authenticate 头", func(t *testing.T) {
		router, w := setupRouter(authSvc)
		req := httptest.NewRequest("POST", "/test", nil)
		req.RemoteAddr = "203.0.113.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
		wa := w.Header().Get("WWW-Authenticate")
		if wa == "" {
			t.Error("401 响应应包含 WWW-Authenticate 头")
		}
		if !strings.Contains(wa, "Bearer") {
			t.Errorf("WWW-Authenticate 应包含 Bearer, got %s", wa)
		}
		if !strings.Contains(wa, "resource_metadata") {
			t.Errorf("WWW-Authenticate 应包含 resource_metadata, got %s", wa)
		}
		if !strings.Contains(wa, ".well-known/oauth-protected-resource") {
			t.Errorf("WWW-Authenticate 应指向 well-known 端点, got %s", wa)
		}
	})

	t.Run("本地回环免认证时不设置 WWW-Authenticate 头", func(t *testing.T) {
		router, w := setupRouter(authSvc)
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "127.0.0.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		wa := w.Header().Get("WWW-Authenticate")
		if wa != "" {
			t.Errorf("回环免认证不应设置 WWW-Authenticate, got %s", wa)
		}
	})
}
