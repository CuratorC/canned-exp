package middlewares

import (
	"net/http"
	"net/http/httptest"
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
		APIKey:     "test-api-key",
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

	t.Run("API Key 不被中间件识别（仅 session token 有效）", func(t *testing.T) {
		router, w := setupRouter(authSvc)
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.RemoteAddr = "203.0.113.1:12345"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("API Key 不应被中间件放行，status = %d", w.Code)
		}
	})

	t.Run("过期 token 返回 401", func(t *testing.T) {
		expiredAuthSvc := auth.NewAuth(auth.Config{
			TOTPSecret: secret,
			APIKey:     "test-api-key",
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
}
