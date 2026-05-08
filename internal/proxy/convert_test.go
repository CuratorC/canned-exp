package proxy

import (
	"encoding/json"
	"testing"
)

// --- 辅助函数 ---

// mustRaw 将 v 序列化为 json.RawMessage，失败则 fatal
func mustRaw(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return json.RawMessage(b)
}

// --- 请求转换测试（Anthropic → OpenAI）---

func TestConvertRequest_BasicMessage(t *testing.T) {
	req := &AnthropicRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 1024,
		Messages: []AnthropicMessage{
			{Role: "user", Content: mustRaw(t, "Hello")},
		},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("messages count = %d, want 1", len(result.Messages))
	}
	if result.Messages[0].Role != "user" {
		t.Errorf("role = %q, want %q", result.Messages[0].Role, "user")
	}
	if result.Messages[0].Content != "Hello" {
		t.Errorf("content = %q, want %q", result.Messages[0].Content, "Hello")
	}
	if result.MaxTokens != 1024 {
		t.Errorf("max_tokens = %d, want 1024", result.MaxTokens)
	}
}

func TestConvertRequest_SystemPrompt(t *testing.T) {
	req := &AnthropicRequest{
		System: mustRaw(t, "You are helpful"),
		Messages: []AnthropicMessage{
			{Role: "user", Content: mustRaw(t, "Hello")},
		},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	if len(result.Messages) != 2 {
		t.Fatalf("messages count = %d, want 2 (system + user)", len(result.Messages))
	}
	if result.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want %q", result.Messages[0].Role, "system")
	}
	if result.Messages[0].Content != "You are helpful" {
		t.Errorf("system content = %q, want %q", result.Messages[0].Content, "You are helpful")
	}
}

func TestConvertRequest_SystemPromptArray(t *testing.T) {
	req := &AnthropicRequest{
		System: mustRaw(t, []map[string]string{
			{"type": "text", "text": "You are helpful"},
			{"type": "text", "text": " Be concise"},
		}),
		Messages: []AnthropicMessage{
			{Role: "user", Content: mustRaw(t, "Hello")},
		},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	if result.Messages[0].Role != "system" {
		t.Errorf("role = %q, want %q", result.Messages[0].Role, "system")
	}
	expected := "You are helpful Be concise"
	if result.Messages[0].Content != expected {
		t.Errorf("content = %q, want %q", result.Messages[0].Content, expected)
	}
}

func TestConvertRequest_ToolDefinitions(t *testing.T) {
	req := &AnthropicRequest{
		Messages: []AnthropicMessage{
			{Role: "user", Content: mustRaw(t, "list files")},
		},
		Tools: []AnthropicTool{
			{
				Name:        "bash",
				Description: "Run a bash command",
				InputSchema: mustRaw(t, map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]string{"type": "string"},
					},
				}),
			},
		},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("tools count = %d, want 1", len(result.Tools))
	}
	tool := result.Tools[0]
	if tool.Type != "function" {
		t.Errorf("tool type = %q, want %q", tool.Type, "function")
	}
	if tool.Function.Name != "bash" {
		t.Errorf("name = %q, want %q", tool.Function.Name, "bash")
	}
	if tool.Function.Description != "Run a bash command" {
		t.Errorf("description = %q, want %q", tool.Function.Description, "Run a bash command")
	}
	if tool.Function.Parameters == nil {
		t.Error("parameters should not be nil")
	}
}

func TestConvertRequest_ToolChoice(t *testing.T) {
	tests := []struct {
		name     string
		choice   AnthropicToolChoice
		wantJSON string
	}{
		{"auto", AnthropicToolChoice{Type: "auto"}, `"auto"`},
		{"any -> required", AnthropicToolChoice{Type: "any"}, `"required"`},
		{"none", AnthropicToolChoice{Type: "none"}, `"none"`},
		{"named", AnthropicToolChoice{Type: "tool", Name: "bash"}, ``}, // 验证结构，不比较字符串
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &AnthropicRequest{
				Messages:   []AnthropicMessage{{Role: "user", Content: mustRaw(t, "hi")}},
				ToolChoice: &tt.choice,
			}
			result, err := ConvertRequest(req, "test-model", false)
			if err != nil {
				t.Fatalf("ConvertRequest() error: %v", err)
			}
			got, _ := json.Marshal(result.ToolChoice)
			if tt.wantJSON != "" {
				if string(got) != tt.wantJSON {
					t.Errorf("tool_choice = %s, want %s", got, tt.wantJSON)
				}
			} else {
				// 结构化比较：解析后验证字段
				var parsed map[string]interface{}
				json.Unmarshal(got, &parsed)
				if parsed["type"] != "function" {
					t.Errorf("tool_choice.type = %v, want function", parsed["type"])
				}
				fn, _ := parsed["function"].(map[string]interface{})
				if fn["name"] != tt.choice.Name {
					t.Errorf("tool_choice.function.name = %v, want %s", fn["name"], tt.choice.Name)
				}
			}
		})
	}
}

func TestConvertRequest_AssistantWithToolUse(t *testing.T) {
	req := &AnthropicRequest{
		Messages: []AnthropicMessage{
			{
				Role: "assistant",
				Content: mustRaw(t, []map[string]interface{}{
					{"type": "text", "text": "I'll run that command."},
					{"type": "tool_use", "id": "toolu_abc123", "name": "bash", "input": map[string]string{"command": "ls"}},
				}),
			},
		},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("messages count = %d, want 1", len(result.Messages))
	}
	msg := result.Messages[0]
	if msg.Role != "assistant" {
		t.Errorf("role = %q, want %q", msg.Role, "assistant")
	}
	if msg.Content != "I'll run that command." {
		t.Errorf("content = %q, want %q", msg.Content, "I'll run that command.")
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("tool_calls count = %d, want 1", len(msg.ToolCalls))
	}
	tc := msg.ToolCalls[0]
	if tc.ID != "toolu_abc123" {
		t.Errorf("id = %q, want %q", tc.ID, "toolu_abc123")
	}
	if tc.Function.Name != "bash" {
		t.Errorf("name = %q, want %q", tc.Function.Name, "bash")
	}
	if tc.Function.Arguments != `{"command":"ls"}` {
		t.Errorf("arguments = %q, want %q", tc.Function.Arguments, `{"command":"ls"}`)
	}
}

func TestConvertRequest_UserWithToolResult(t *testing.T) {
	req := &AnthropicRequest{
		Messages: []AnthropicMessage{
			{
				Role: "user",
				Content: mustRaw(t, []map[string]interface{}{
					{"type": "tool_result", "tool_use_id": "toolu_abc123", "content": "file1.txt\nfile2.txt"},
				}),
			},
		},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	if len(result.Messages) != 1 {
		t.Fatalf("messages count = %d, want 1", len(result.Messages))
	}
	msg := result.Messages[0]
	if msg.Role != "tool" {
		t.Errorf("role = %q, want %q", msg.Role, "tool")
	}
	if msg.ToolCallID != "toolu_abc123" {
		t.Errorf("tool_call_id = %q, want %q", msg.ToolCallID, "toolu_abc123")
	}
	if msg.Content != "file1.txt\nfile2.txt" {
		t.Errorf("content = %q, want %q", msg.Content, "file1.txt\nfile2.txt")
	}
}

func TestConvertRequest_MixedContentBlocks(t *testing.T) {
	// user 消息中同时有 text 和 tool_result
	req := &AnthropicRequest{
		Messages: []AnthropicMessage{
			{
				Role: "user",
				Content: mustRaw(t, []map[string]interface{}{
					{"type": "text", "text": "Here is the result:"},
					{"type": "tool_result", "tool_use_id": "toolu_abc123", "content": "output"},
				}),
			},
		},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	// text + tool_result 应拆分为 user + tool 两条消息
	if len(result.Messages) != 2 {
		t.Fatalf("messages count = %d, want 2", len(result.Messages))
	}
	if result.Messages[0].Role != "user" {
		t.Errorf("first role = %q, want %q", result.Messages[0].Role, "user")
	}
	if result.Messages[0].Content != "Here is the result:" {
		t.Errorf("first content = %q, want %q", result.Messages[0].Content, "Here is the result:")
	}
	if result.Messages[1].Role != "tool" {
		t.Errorf("second role = %q, want %q", result.Messages[1].Role, "tool")
	}
}

func TestModelName_MappingAndFallback(t *testing.T) {
	svc := NewProxyService(Config{
		ModelMap:     `{"claude-sonnet-4-20250514":"gpt-4o","claude-opus-4-6":"gpt-4o"}`,
		DefaultModel: "DeepSeek V4 Pro",
	})

	tests := []struct {
		input string
		want  string
	}{
		{"claude-sonnet-4-20250514", "gpt-4o"},        // 精确映射
		{"claude-opus-4-6", "gpt-4o"},                  // 精确映射
		{"claude-opus-4-7", "DeepSeek V4 Pro"},          // 未映射 → 回退到 DefaultModel
		{"claude-sonnet-4-7", "DeepSeek V4 Pro"},        // 未映射 → 回退到 DefaultModel
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := svc.ModelName(tt.input)
			if got != tt.want {
				t.Errorf("ModelName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestModelName_NoFallback(t *testing.T) {
	// 没有 DefaultModel，未映射的模型名原样传递
	svc := NewProxyService(Config{
		ModelMap:     `{"claude-sonnet-4-20250514":"gpt-4o"}`,
		DefaultModel: "",
	})
	got := svc.ModelName("claude-opus-4-7")
	if got != "claude-opus-4-7" {
		t.Errorf("ModelName with no fallback = %q, want %q", got, "claude-opus-4-7")
	}
}

func TestConvertRequest_StopSequences(t *testing.T) {
	req := &AnthropicRequest{
		Messages:      []AnthropicMessage{{Role: "user", Content: mustRaw(t, "hi")}},
		StopSequences: []string{"\n\nHuman:", "---END---"},
	}

	result, err := ConvertRequest(req, "test-model", false)
	if err != nil {
		t.Fatalf("ConvertRequest() error: %v", err)
	}

	if len(result.Stop) != 2 {
		t.Fatalf("stop count = %d, want 2", len(result.Stop))
	}
	if result.Stop[0] != "\n\nHuman:" {
		t.Errorf("stop[0] = %q, want %q", result.Stop[0], "\n\nHuman:")
	}
	if result.Stop[1] != "---END---" {
		t.Errorf("stop[1] = %q, want %q", result.Stop[1], "---END---")
	}
}

// --- 响应转换测试（OpenAI → Anthropic）---

func TestConvertResponse_BasicText(t *testing.T) {
	openaiResp := &OpenAIResponse{
		ID:    "chatcmpl-123",
		Model: "gpt-4o",
		Choices: []OpenAIChoice{
			{
				Index:        0,
				Message:      OpenAIMessage{Role: "assistant", Content: "Hello! How can I help you?"},
				FinishReason: "stop",
			},
		},
		Usage: OpenAIUsage{PromptTokens: 10, CompletionTokens: 7},
	}

	result := ConvertResponse(openaiResp, "claude-sonnet-4-20250514", "msg_test123")

	if result.ID != "msg_test123" {
		t.Errorf("id = %q, want %q", result.ID, "msg_test123")
	}
	if result.Type != "message" {
		t.Errorf("type = %q, want %q", result.Type, "message")
	}
	if result.Role != "assistant" {
		t.Errorf("role = %q, want %q", result.Role, "assistant")
	}
	if result.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want original model name", result.Model)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content blocks = %d, want 1", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("content[0].type = %q, want %q", result.Content[0].Type, "text")
	}
	if result.Content[0].Text != "Hello! How can I help you?" {
		t.Errorf("content[0].text = %q, want %q", result.Content[0].Text, "Hello! How can I help you?")
	}
	if result.StopReason != "end_turn" {
		t.Errorf("stop_reason = %q, want %q", result.StopReason, "end_turn")
	}
	if result.Usage.InputTokens != 10 {
		t.Errorf("input_tokens = %d, want 10", result.Usage.InputTokens)
	}
	if result.Usage.OutputTokens != 7 {
		t.Errorf("output_tokens = %d, want 7", result.Usage.OutputTokens)
	}
}

func TestConvertResponse_ToolCalls(t *testing.T) {
	openaiResp := &OpenAIResponse{
		ID: "chatcmpl-123",
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "I'll check that.",
					ToolCalls: []OpenAIToolCall{
						{
							ID:   "call_abc",
							Type: "function",
							Function: OpenAIFunction{
								Name:      "bash",
								Arguments: `{"command":"ls -la"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			},
		},
		Usage: OpenAIUsage{PromptTokens: 20, CompletionTokens: 15},
	}

	result := ConvertResponse(openaiResp, "claude-sonnet-4-20250514", "msg_test")

	// text + tool_use = 2 个 content blocks
	if len(result.Content) != 2 {
		t.Fatalf("content blocks = %d, want 2", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("content[0].type = %q, want %q", result.Content[0].Type, "text")
	}
	if result.Content[1].Type != "tool_use" {
		t.Errorf("content[1].type = %q, want %q", result.Content[1].Type, "tool_use")
	}
	if result.Content[1].ID != "call_abc" {
		t.Errorf("content[1].id = %q, want %q", result.Content[1].ID, "call_abc")
	}
	if result.Content[1].Name != "bash" {
		t.Errorf("content[1].name = %q, want %q", result.Content[1].Name, "bash")
	}
	if result.StopReason != "tool_use" {
		t.Errorf("stop_reason = %q, want %q", result.StopReason, "tool_use")
	}
}

func TestConvertResponse_EmptyContentWithToolCalls(t *testing.T) {
	// OpenAI 返回空 content + tool_calls，不应生成空 text block
	openaiResp := &OpenAIResponse{
		Choices: []OpenAIChoice{
			{
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: "",
					ToolCalls: []OpenAIToolCall{
						{ID: "call_1", Type: "function", Function: OpenAIFunction{Name: "bash", Arguments: "{}"}},
					},
				},
				FinishReason: "tool_calls",
			},
		},
	}

	result := ConvertResponse(openaiResp, "model", "msg_test")

	if len(result.Content) != 1 {
		t.Fatalf("content blocks = %d, want 1 (only tool_use)", len(result.Content))
	}
	if result.Content[0].Type != "tool_use" {
		t.Errorf("content[0].type = %q, want %q", result.Content[0].Type, "tool_use")
	}
}

func TestConvertResponse_FinishReasons(t *testing.T) {
	tests := []struct {
		openai string
		want   string
	}{
		{"stop", "end_turn"},
		{"tool_calls", "tool_use"},
		{"length", "max_tokens"},
		{"content_filter", "end_turn"},
	}

	for _, tt := range tests {
		t.Run(tt.openai+" -> "+tt.want, func(t *testing.T) {
			got := mapFinishReason(tt.openai)
			if got != tt.want {
				t.Errorf("mapFinishReason(%q) = %q, want %q", tt.openai, got, tt.want)
			}
		})
	}
}

// --- 错误转换测试 ---

func TestConvertError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantType   string
	}{
		{"400 -> invalid_request_error", 400, "invalid_request_error"},
		{"401 -> authentication_error", 401, "authentication_error"},
		{"403 -> permission_error", 403, "permission_error"},
		{"404 -> not_found_error", 404, "not_found_error"},
		{"429 -> rate_limit_error", 429, "rate_limit_error"},
		{"500 -> api_error", 500, "api_error"},
		{"502 -> api_error", 502, "api_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errBody := []byte(`{"error":{"message":"test error","type":"test_type","code":"test_code"}}`)
			result := ConvertError(errBody, tt.statusCode)

			if result.Type != "error" {
				t.Errorf("outer type = %q, want %q", result.Type, "error")
			}
			if result.Error.Type != tt.wantType {
				t.Errorf("error type = %q, want %q", result.Error.Type, tt.wantType)
			}
			if result.Error.Message != "test error" {
				t.Errorf("message = %q, want %q", result.Error.Message, "test error")
			}
		})
	}
}

func TestConvertError_EmptyBody(t *testing.T) {
	result := ConvertError(nil, 500)
	if result.Type != "error" {
		t.Errorf("type = %q, want %q", result.Type, "error")
	}
	if result.Error.Type != "api_error" {
		t.Errorf("error type = %q, want %q", result.Error.Type, "api_error")
	}
	if result.Error.Message == "" {
		t.Error("message should not be empty")
	}
}
