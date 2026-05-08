package proxy

import "encoding/json"

// OpenAI Chat Completions API 请求/响应类型定义

// OpenAIRequest 表示 OpenAI /v1/chat/completions 请求
type OpenAIRequest struct {
	Model      string          `json:"model"`
	Messages   []OpenAIMessage `json:"messages"`
	MaxTokens  int             `json:"max_tokens,omitempty"`
	Stream     bool            `json:"stream,omitempty"`
	Temperature *float64       `json:"temperature,omitempty"`
	TopP       *float64        `json:"top_p,omitempty"`
	Stop       []string        `json:"stop,omitempty"`
	Tools      []OpenAITool    `json:"tools,omitempty"`
	ToolChoice interface{}     `json:"tool_choice,omitempty"` // string 或 object
}

// OpenAIMessage 表示一条 OpenAI 消息
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// OpenAITool 表示 OpenAI 工具定义
type OpenAITool struct {
	Type     string           `json:"type"` // always "function"
	Function OpenAIFunction   `json:"function"`
}

// OpenAIFunction 表示 OpenAI 函数定义
type OpenAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Arguments   string          `json:"arguments,omitempty"` // tool call 响应中使用
}

// OpenAIToolCall 表示 OpenAI 工具调用
type OpenAIToolCall struct {
	Index    int             `json:"index,omitempty"`
	ID       string          `json:"id"`
	Type     string          `json:"type"` // always "function"
	Function OpenAIFunction  `json:"function"`
}

// OpenAIResponse 表示 OpenAI 非流式响应
type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object,omitempty"`
	Model   string         `json:"model,omitempty"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage,omitempty"`
}

// OpenAIChoice 表示一个选择
type OpenAIChoice struct {
	Index        int            `json:"index"`
	Message      OpenAIMessage  `json:"message"`
	FinishReason string         `json:"finish_reason"`
}

// OpenAIUsage 表示 OpenAI token 用量
type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// --- 流式类型 ---

// OpenAIChunk 表示 OpenAI 流式响应的一个 chunk
type OpenAIChunk struct {
	ID      string            `json:"id"`
	Object  string            `json:"object,omitempty"`
	Choices []OpenAIChunkChoice `json:"choices"`
}

// OpenAIChunkChoice 表示流式选择
type OpenAIChunkChoice struct {
	Index        int          `json:"index"`
	Delta        OpenAIDelta  `json:"delta"`
	FinishReason *string      `json:"finish_reason"`
}

// OpenAIDelta 表示流式增量
type OpenAIDelta struct {
	Role      string                `json:"role,omitempty"`
	Content   string                `json:"content,omitempty"`
	ToolCalls []OpenAIToolCallDelta `json:"tool_calls,omitempty"`
}

// OpenAIToolCallDelta 表示流式工具调用增量
type OpenAIToolCallDelta struct {
	Index    int                `json:"index"`
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function OpenAIFunctionDelta `json:"function"`
}

// OpenAIFunctionDelta 表示流式函数增量
type OpenAIFunctionDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

// OpenAIErrorBody 表示 OpenAI 错误响应体
type OpenAIErrorBody struct {
	Error OpenAIErrorDetail `json:"error"`
}

// OpenAIErrorDetail 表示 OpenAI 错误详情
type OpenAIErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}
