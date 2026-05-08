package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// --- 辅助函数 ---

// setupHandler 创建指向 mock 后端的 ProxyService 和 Gin 路由
func setupHandler(t *testing.T, backend http.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(backend)
	t.Cleanup(server.Close)

	svc := NewProxyService(Config{
		BaseURL:  server.URL,
		APIKey:   "test-key",
		ModelMap: `{"claude-sonnet-4-20250514":"gpt-4o"}`,
	})

	r := gin.New()
	r.POST("/v1/messages", MessagesHandler(svc))
	return r
}

// doPost 向 /v1/messages 发送请求
func doPost(router *gin.Engine, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(string(b)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// --- Handler 测试 ---

func TestMessagesHandler_NonStreaming(t *testing.T) {
	router := setupHandler(t, func(w http.ResponseWriter, r *http.Request) {
		// 验证转发到后端的请求格式
		var openaiReq OpenAIRequest
		if err := json.NewDecoder(r.Body).Decode(&openaiReq); err != nil {
			t.Errorf("backend received invalid body: %v", err)
			w.WriteHeader(400)
			return
		}
		// 验证模型已映射
		if openaiReq.Model != "gpt-4o" {
			t.Errorf("backend model = %q, want %q", openaiReq.Model, "gpt-4o")
		}
		// 验证 Authorization 头
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("auth header = %q, want %q", r.Header.Get("Authorization"), "Bearer test-key")
		}

		resp := OpenAIResponse{
			ID:    "chatcmpl-test",
			Model: "gpt-4o",
			Choices: []OpenAIChoice{
				{
					Message:      OpenAIMessage{Role: "assistant", Content: "Hi there!"},
					FinishReason: "stop",
				},
			},
			Usage: OpenAIUsage{PromptTokens: 5, CompletionTokens: 3},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	w := doPost(router, map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "Hello"}},
	})

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp AnthropicResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.Type != "message" {
		t.Errorf("type = %q, want %q", resp.Type, "message")
	}
	if resp.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want original model name", resp.Model)
	}
	if len(resp.Content) != 1 || resp.Content[0].Text != "Hi there!" {
		t.Errorf("content = %+v, want text 'Hi there!'", resp.Content)
	}
}

func TestMessagesHandler_Streaming(t *testing.T) {
	router := setupHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)

		chunks := []string{
			`{"id":"chatcmpl-1","choices":[{"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-1","choices":[{"delta":{"content":"Hi!"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-1","choices":[{"delta":{},"finish_reason":"stop"}]}`,
		}
		for _, c := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", c)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	})

	w := doPost(router, map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "Hello"}},
		"stream":     true,
	})

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("content-type = %q, want text/event-stream", ct)
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: message_start") {
		t.Error("missing message_start event")
	}
	if !strings.Contains(body, "event: message_stop") {
		t.Error("missing message_stop event")
	}
	if !strings.Contains(body, "content_block_delta") {
		t.Error("missing content_block_delta event")
	}
}

func TestMessagesHandler_BackendError(t *testing.T) {
	router := setupHandler(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "Rate limit exceeded",
				"type":    "rate_limit_error",
				"code":    "rate_limit_exceeded",
			},
		})
	})

	w := doPost(router, map[string]interface{}{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "Hello"}},
	})

	if w.Code != 429 {
		t.Fatalf("status = %d, want 429", w.Code)
	}

	var errResp AnthropicError
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if errResp.Type != "error" {
		t.Errorf("outer type = %q, want %q", errResp.Type, "error")
	}
	if errResp.Error.Type != "rate_limit_error" {
		t.Errorf("error type = %q, want %q", errResp.Error.Type, "rate_limit_error")
	}
	if errResp.Error.Message != "Rate limit exceeded" {
		t.Errorf("message = %q, want %q", errResp.Error.Message, "Rate limit exceeded")
	}
}

func TestMessagesHandler_InvalidRequest(t *testing.T) {
	router := setupHandler(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("backend should not be called for invalid request")
	})

	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader("not json at all"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Fatalf("status = %d, want 400", w.Code)
	}

	var errResp AnthropicError
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Type != "error" {
		t.Errorf("type = %q, want %q", errResp.Type, "error")
	}
	if errResp.Error.Type != "invalid_request_error" {
		t.Errorf("error type = %q, want %q", errResp.Error.Type, "invalid_request_error")
	}
}

func TestMessagesHandler_BackendUnreachable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewProxyService(Config{
		BaseURL: "http://127.0.0.1:1", // 不可达端口
		APIKey:  "test",
	})
	r := gin.New()
	r.POST("/v1/messages", MessagesHandler(svc))

	w := doPost(r, map[string]interface{}{
		"model":      "test-model",
		"max_tokens": 100,
		"messages":   []map[string]string{{"role": "user", "content": "Hello"}},
	})

	if w.Code != 502 {
		t.Fatalf("status = %d, want 502", w.Code)
	}

	var errResp AnthropicError
	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error.Type != "api_error" {
		t.Errorf("error type = %q, want %q", errResp.Error.Type, "api_error")
	}
}
