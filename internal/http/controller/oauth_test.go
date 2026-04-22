package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"canned-exp/internal/auth"
	"github.com/gin-gonic/gin"
)

const testBaseURL = "http://testserver:3100"

func setupOAuthTest(t *testing.T) (*auth.Auth, *OAuthController) {
	t.Helper()
	secret := generateTestSecret(t)
	authSvc := auth.NewAuth(auth.Config{
		TOTPSecret: secret,
		SessionTTL: time.Hour,
	})
	ctrl := NewOAuthController(authSvc, testBaseURL)
	return authSvc, ctrl
}

func setupOAuthRouter(authSvc *auth.Auth, ctrl *OAuthController) (*gin.Engine, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/.well-known/oauth-protected-resource", ctrl.ProtectedResourceMetadata)
	r.GET("/.well-known/oauth-authorization-server", ctrl.AuthorizationServerMetadata)
	r.POST("/oauth/register", ctrl.Register)
	r.GET("/oauth/authorize", ctrl.AuthorizeGet)
	r.POST("/oauth/authorize", ctrl.AuthorizePost)
	r.POST("/oauth/token", ctrl.Token)
	return r, httptest.NewRecorder()
}

// --- 测试用例 1: /.well-known/oauth-protected-resource ---

func TestOAuth_ProtectedResourceMetadata(t *testing.T) {
	_, ctrl := setupOAuthTest(t)
	router, w := setupOAuthRouter(nil, ctrl)

	req := httptest.NewRequest("GET", "/.well-known/oauth-protected-resource", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	servers, ok := resp["authorization_servers"].([]interface{})
	if !ok || len(servers) == 0 || servers[0] != testBaseURL {
		t.Errorf("authorization_servers 应包含 %s", testBaseURL)
	}
	if resp["resource"] != testBaseURL {
		t.Errorf("resource = %v, want %s", resp["resource"], testBaseURL)
	}
	scopes, ok := resp["scopes_supported"].([]interface{})
	if !ok || len(scopes) == 0 || scopes[0] != "mcp" {
		t.Error("scopes_supported 应包含 mcp")
	}
	methods, ok := resp["bearer_methods_supported"].([]interface{})
	if !ok || len(methods) == 0 || methods[0] != "header" {
		t.Error("bearer_methods_supported 应包含 header")
	}
}

// --- 测试用例 2: /.well-known/oauth-authorization-server ---

func TestOAuth_AuthorizationServerMetadata(t *testing.T) {
	_, ctrl := setupOAuthTest(t)
	router, w := setupOAuthRouter(nil, ctrl)

	req := httptest.NewRequest("GET", "/.well-known/oauth-authorization-server", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["issuer"] != testBaseURL {
		t.Errorf("issuer = %v, want %s", resp["issuer"], testBaseURL)
	}
	if resp["authorization_endpoint"] != testBaseURL+"/oauth/authorize" {
		t.Errorf("authorization_endpoint = %v", resp["authorization_endpoint"])
	}
	if resp["token_endpoint"] != testBaseURL+"/oauth/token" {
		t.Errorf("token_endpoint = %v", resp["token_endpoint"])
	}
	if resp["registration_endpoint"] != testBaseURL+"/oauth/register" {
		t.Errorf("registration_endpoint = %v", resp["registration_endpoint"])
	}

	responseTypes, _ := resp["response_types_supported"].([]interface{})
	if len(responseTypes) == 0 || responseTypes[0] != "code" {
		t.Error("response_types_supported 应包含 code")
	}
	challengeMethods, _ := resp["code_challenge_methods_supported"].([]interface{})
	if len(challengeMethods) == 0 || challengeMethods[0] != "S256" {
		t.Error("code_challenge_methods_supported 应包含 S256")
	}
	grantTypes, _ := resp["grant_types_supported"].([]interface{})
	if len(grantTypes) == 0 || grantTypes[0] != "authorization_code" {
		t.Error("grant_types_supported 应包含 authorization_code")
	}
}

// --- 测试用例 3: /oauth/register ---

func TestOAuth_Register(t *testing.T) {
	_, ctrl := setupOAuthTest(t)
	router, w := setupOAuthRouter(nil, ctrl)

	t.Run("正常注册返回 201 + client_id + client_secret", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"client_name":   "Claude Code",
			"redirect_uris": []string{"http://localhost:8080/callback"},
		})
		req := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", w.Code)
		}
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		
		if resp["client_id"] == "" {
			t.Error("应返回 client_id")
		}
		if resp["client_secret"] == "" {
			t.Error("应返回 client_secret")
		}
		if resp["client_name"] != "Claude Code" {
			t.Errorf("client_name = %v, want Claude Code", resp["client_name"])
		}
	})

	t.Run("缺少 redirect_uris 返回 400", func(t *testing.T) {
		w2 := httptest.NewRecorder()
		body, _ := json.Marshal(map[string]interface{}{
			"client_name": "test",
		})
		req := httptest.NewRequest("POST", "/oauth/register", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w2, req)

		if w2.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w2.Code)
		}
	})
}

// --- 测试用例 4: /oauth/authorize GET ---

func TestOAuth_AuthorizeGet(t *testing.T) {
	authSvc, ctrl := setupOAuthTest(t)
	router, _ := setupOAuthRouter(authSvc, ctrl)

	// 先注册一个客户端
	client := authSvc.RegisterClient("test-app", []string{"http://localhost:9090/callback"})

	t.Run("合法参数返回 200 HTML 页面", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id="+client.ClientID+
			"&redirect_uri="+url.QueryEscape("http://localhost:9090/callback")+
			"&code_challenge=test_challenge&state=abc", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "totp_code") {
			t.Error("页面应包含 TOTP 输入框")
		}
		if !strings.Contains(body, "hidden") {
			t.Error("页面应包含 hidden 字段")
		}
	})

	t.Run("缺少 client_id 返回 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/oauth/authorize?redirect_uri=http://localhost&code_challenge=x", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("未注册的 client_id 返回 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id=nonexistent&redirect_uri=http://localhost&code_challenge=x", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})

	t.Run("redirect_uri 不匹配返回 400", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/oauth/authorize?client_id="+client.ClientID+
			"&redirect_uri="+url.QueryEscape("http://evil.com/callback")+
			"&code_challenge=x", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}

// --- 测试用例 5: /oauth/authorize POST ---

func TestOAuth_AuthorizePost(t *testing.T) {
	secret := generateTestSecret(t)

	authSvc := auth.NewAuth(auth.Config{TOTPSecret: secret, SessionTTL: time.Hour})
	ctrl := NewOAuthController(authSvc, testBaseURL)
	router, _ := setupOAuthRouter(authSvc, ctrl)
	client := authSvc.RegisterClient("post-test", []string{"http://localhost:9090/callback"})

	t.Run("TOTP 错误重新渲染页面并提示", func(t *testing.T) {
		w := httptest.NewRecorder()
		form := url.Values{
			"client_id":      {client.ClientID},
			"redirect_uri":   {"http://localhost:9090/callback"},
			"code_challenge": {"test_challenge"},
			"state":          {"mystate"},
			"totp_code":      {"000000"},
		}
		req := httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (重新渲染)", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "验证码无效") {
			t.Error("错误页面应包含提示信息")
		}
	})

	t.Run("TOTP 正确 302 重定向到 redirect_uri 并带 code 和 state", func(t *testing.T) {
		w := httptest.NewRecorder()
		code := generateTestCode(t, secret)
		form := url.Values{
			"client_id":      {client.ClientID},
			"redirect_uri":   {"http://localhost:9090/callback"},
			"code_challenge": {"test_challenge"},
			"state":          {"mystate"},
			"totp_code":      {code},
		}
		req := httptest.NewRequest("POST", "/oauth/authorize", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// 禁止自动跟随重定向
		router.ServeHTTP(w, req)

		if w.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302", w.Code)
		}
		loc := w.Header().Get("Location")
		if loc == "" {
			t.Fatal("应有 Location 头")
		}
		parsed, _ := url.Parse(loc)
		if parsed.Query().Get("code") == "" {
			t.Error("重定向 URL 应包含 code 参数")
		}
		if parsed.Query().Get("state") != "mystate" {
			t.Error("重定向 URL 应包含原始 state 参数")
		}
	})
}

// --- 测试用例 6: /oauth/token ---

func TestOAuth_Token(t *testing.T) {
	secret := generateTestSecret(t)
	authSvc := auth.NewAuth(auth.Config{TOTPSecret: secret, SessionTTL: time.Hour})
	ctrl := NewOAuthController(authSvc, testBaseURL)
	router, _ := setupOAuthRouter(authSvc, ctrl)

	// 先注册客户端并生成授权码
	client := authSvc.RegisterClient("token-test", []string{"http://localhost:9090/callback"})
	authCode := authSvc.CreateAuthorizationCode(client.ClientID, "http://localhost:9090/callback", "", "oauth:svc")

	t.Run("正常交换返回 access_token", func(t *testing.T) {
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode},
			"client_id":     {client.ClientID},
			"client_secret": {client.ClientSecret},
		}
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200, body = %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["access_token"] == "" {
			t.Error("应返回 access_token")
		}
		if resp["token_type"] != "bearer" {
			t.Errorf("token_type = %v, want bearer", resp["token_type"])
		}
		if resp["expires_in"] == nil {
			t.Error("应返回 expires_in")
		}
	})

	t.Run("已使用的 code 返回 400 invalid_grant", func(t *testing.T) {
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {authCode}, // 已被上面的测试使用
			"client_id":     {client.ClientID},
			"client_secret": {client.ClientSecret},
		}
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["error"] != "invalid_grant" {
			t.Errorf("error = %v, want invalid_grant", resp["error"])
		}
	})

	t.Run("错误的 client_secret 返回 401 invalid_client", func(t *testing.T) {
		code := authSvc.CreateAuthorizationCode(client.ClientID, "http://localhost:9090/callback", "", "svc")
		form := url.Values{
			"grant_type":    {"authorization_code"},
			"code":          {code},
			"client_id":     {client.ClientID},
			"client_secret": {"wrong-secret"},
		}
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", w.Code)
		}
	})

	t.Run("不支持的 grant_type 返回 400", func(t *testing.T) {
		form := url.Values{
			"grant_type":    {"client_credentials"},
			"code":          {"fake"},
			"client_id":     {client.ClientID},
			"client_secret": {client.ClientSecret},
		}
		req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", w.Code)
		}
	})
}
