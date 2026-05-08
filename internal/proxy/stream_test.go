package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// --- SSE 解析辅助 ---

// sseEvent 表示一个解析后的 SSE 事件
type sseEvent struct {
	Event string
	Data  string
}

// parseSSE 从响应体解析所有 SSE 事件
func parseSSE(t *testing.T, body string) []sseEvent {
	t.Helper()
	var events []sseEvent
	scanner := bufio.NewScanner(strings.NewReader(body))
	var curEvent string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			curEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			events = append(events, sseEvent{
				Event: curEvent,
				Data:  strings.TrimPrefix(line, "data: "),
			})
			curEvent = ""
		}
	}
	return events
}

// --- Mock 服务器辅助 ---

// mockOpenAISSE 创建返回 OpenAI SSE 格式数据的 mock 服务器
func mockOpenAISSE(t *testing.T, chunks ...string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		fmt.Fprintf(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
}

// runStreamProxy 执行流式代理并返回解析后的事件
func runStreamProxy(t *testing.T, resp *http.Response) []sseEvent {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/messages", nil)

	StreamProxy(c, resp, "msg_test", "claude-sonnet-4-20250514")

	return parseSSE(t, w.Body.String())
}

// --- 流式测试 ---

func TestStreamProxy_TextOnly(t *testing.T) {
	chunks := []string{
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
	}

	server := mockOpenAISSE(t, chunks...)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("get mock response: %v", err)
	}
	defer resp.Body.Close()

	events := runStreamProxy(t, resp)

	// 期望事件序列：message_start → content_block_start → 2x content_block_delta → content_block_stop → message_delta → message_stop
	expectedEvents := []string{
		"message_start",
		"content_block_start",
		"content_block_delta",
		"content_block_delta",
		"content_block_stop",
		"message_delta",
		"message_stop",
	}

	if len(events) != len(expectedEvents) {
		t.Fatalf("events count = %d, want %d", len(events), len(expectedEvents))
	}

	for i, expected := range expectedEvents {
		if events[i].Event != expected {
			t.Errorf("event[%d] = %q, want %q", i, events[i].Event, expected)
		}
	}

	// 验证 delta 文本拼接
	var textParts []string
	for _, e := range events {
		if e.Event == "content_block_delta" {
			var data map[string]interface{}
			json.Unmarshal([]byte(e.Data), &data)
			if delta, ok := data["delta"].(map[string]interface{}); ok {
				if text, ok := delta["text"].(string); ok {
					textParts = append(textParts, text)
				}
			}
		}
	}
	if len(textParts) != 2 || textParts[0] != "Hello" || textParts[1] != " world" {
		t.Errorf("text parts = %v, want [Hello, \" world\"]", textParts)
	}
}

func TestStreamProxy_ToolCalls(t *testing.T) {
	chunks := []string{
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"bash","arguments":""}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"command\":\"ls\"}"}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
	}

	server := mockOpenAISSE(t, chunks...)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("get mock response: %v", err)
	}
	defer resp.Body.Close()

	events := runStreamProxy(t, resp)

	// 验证存在 tool_use 的 content_block_start
	foundToolStart := false
	for _, e := range events {
		if e.Event == "content_block_start" {
			var data map[string]interface{}
			json.Unmarshal([]byte(e.Data), &data)
			if block, ok := data["content_block"].(map[string]interface{}); ok {
				if block["type"] == "tool_use" {
					foundToolStart = true
					if block["id"] != "call_abc" {
						t.Errorf("tool_use id = %v, want call_abc", block["id"])
					}
					if block["name"] != "bash" {
						t.Errorf("tool_use name = %v, want bash", block["name"])
					}
				}
			}
		}
	}
	if !foundToolStart {
		t.Error("未找到 tool_use content_block_start")
	}

	// 验证 stop_reason 为 tool_use
	for _, e := range events {
		if e.Event == "message_delta" {
			var data map[string]interface{}
			json.Unmarshal([]byte(e.Data), &data)
			if delta, ok := data["delta"].(map[string]interface{}); ok {
				if delta["stop_reason"] != "tool_use" {
					t.Errorf("stop_reason = %v, want tool_use", delta["stop_reason"])
				}
			}
		}
	}
}

func TestStreamProxy_MixedTextAndToolCalls(t *testing.T) {
	chunks := []string{
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"I'll run that."},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"bash","arguments":""}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{}"}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
	}

	server := mockOpenAISSE(t, chunks...)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("get mock response: %v", err)
	}
	defer resp.Body.Close()

	events := runStreamProxy(t, resp)

	// 应有两个 content_block_start：text(index 0) + tool_use(index 1)
	var blockTypes []string
	for _, e := range events {
		if e.Event == "content_block_start" {
			var data map[string]interface{}
			json.Unmarshal([]byte(e.Data), &data)
			if block, ok := data["content_block"].(map[string]interface{}); ok {
				blockTypes = append(blockTypes, block["type"].(string))
			}
		}
	}
	if len(blockTypes) != 2 {
		t.Fatalf("content_block_start count = %d, want 2", len(blockTypes))
	}
	if blockTypes[0] != "text" {
		t.Errorf("first block = %q, want %q", blockTypes[0], "text")
	}
	if blockTypes[1] != "tool_use" {
		t.Errorf("second block = %q, want %q", blockTypes[1], "tool_use")
	}
}

func TestStreamProxy_MultipleToolCalls(t *testing.T) {
	chunks := []string{
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"bash","arguments":""}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"command\":\"ls\"}"}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"id":"call_2","type":"function","function":{"name":"read","arguments":""}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{\"path\":\"file.txt\"}"}}]},"finish_reason":null}]}`,
		`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
	}

	server := mockOpenAISSE(t, chunks...)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("get mock response: %v", err)
	}
	defer resp.Body.Close()

	events := runStreamProxy(t, resp)

	// 应有两个 tool_use content_block_start
	var toolNames []string
	for _, e := range events {
		if e.Event == "content_block_start" {
			var data map[string]interface{}
			json.Unmarshal([]byte(e.Data), &data)
			if block, ok := data["content_block"].(map[string]interface{}); ok {
				if block["type"] == "tool_use" {
					toolNames = append(toolNames, block["name"].(string))
				}
			}
		}
	}
	if len(toolNames) != 2 {
		t.Fatalf("tool_use blocks = %d, want 2", len(toolNames))
	}
	if toolNames[0] != "bash" {
		t.Errorf("tool[0] = %q, want %q", toolNames[0], "bash")
	}
	if toolNames[1] != "read" {
		t.Errorf("tool[1] = %q, want %q", toolNames[1], "read")
	}
}

func TestStreamProxy_FinishReasonMapping(t *testing.T) {
	tests := []struct {
		name     string
		finish   string
		wantStop string
	}{
		{"stop -> end_turn", `"stop"`, "end_turn"},
		{"tool_calls -> tool_use", `"tool_calls"`, "tool_use"},
		{"length -> max_tokens", `"length"`, "max_tokens"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := []string{
				`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
				`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}`,
				`{"id":"chatcmpl-1","choices":[{"index":0,"delta":{},"finish_reason":` + tt.finish + `}]}`,
			}

			server := mockOpenAISSE(t, chunks...)
			defer server.Close()

			resp, err := http.Get(server.URL)
			if err != nil {
				t.Fatalf("get mock response: %v", err)
			}
			defer resp.Body.Close()

			events := runStreamProxy(t, resp)

			// 从 message_delta 中提取 stop_reason
			for _, e := range events {
				if e.Event == "message_delta" {
					var data map[string]interface{}
					json.Unmarshal([]byte(e.Data), &data)
					if delta, ok := data["delta"].(map[string]interface{}); ok {
						got, _ := delta["stop_reason"].(string)
						if got != tt.wantStop {
							t.Errorf("stop_reason = %q, want %q", got, tt.wantStop)
						}
						return
					}
				}
			}
			t.Fatal("未找到 message_delta 事件")
		})
	}
}
