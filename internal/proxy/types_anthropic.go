package proxy

import "encoding/json"

// Anthropic Messages API 请求/响应类型定义

// AnthropicRequest 表示 Anthropic /v1/messages 请求
type AnthropicRequest struct {
	Model         string               `json:"model"`
	MaxTokens     int                  `json:"max_tokens"`
	Messages      []AnthropicMessage   `json:"messages"`
	System        json.RawMessage      `json:"system,omitempty"`        // string 或 []AnthropicSystemBlock
	Stream        bool                 `json:"stream,omitempty"`
	Temperature   *float64             `json:"temperature,omitempty"`
	TopP          *float64             `json:"top_p,omitempty"`
	TopK          *int                 `json:"top_k,omitempty"`
	StopSequences []string             `json:"stop_sequences,omitempty"`
	Tools         []AnthropicTool      `json:"tools,omitempty"`
	ToolChoice    *AnthropicToolChoice `json:"tool_choice,omitempty"`
	Metadata      interface{}          `json:"metadata,omitempty"`
}

// AnthropicMessage 表示一条消息，Content 为 json.RawMessage 以支持字符串和数组两种格式
type AnthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// ContentBlocks 将 Content 解析为内容块数组
func (m *AnthropicMessage) ContentBlocks() ([]AnthropicContentBlock, error) {
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(m.Content, &s); err == nil {
		return []AnthropicContentBlock{{Type: "text", Text: s}}, nil
	}
	// 解析为数组
	var blocks []AnthropicContentBlock
	if err := json.Unmarshal(m.Content, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// AnthropicContentBlock 表示内容块（text、tool_use、tool_result 等）
type AnthropicContentBlock struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	Content    interface{}     `json:"content,omitempty"` // tool_result 的 content，可以是 string 或 array
	Source     interface{}     `json:"source,omitempty"`  // image block 的 source
	Thinking   string          `json:"thinking,omitempty"`
	Signature  string          `json:"signature,omitempty"`
}

// AnthropicTool 表示 Anthropic 工具定义
type AnthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// AnthropicToolChoice 表示工具选择策略
type AnthropicToolChoice struct {
	Type string `json:"type"` // auto, any, none, tool
	Name string `json:"name,omitempty"`
}

// AnthropicSystemBlock 用于解析 system 字段为数组的情况
type AnthropicSystemBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// AnthropicResponse 表示 Anthropic /v1/messages 非流式响应
type AnthropicResponse struct {
	ID         string                   `json:"id"`
	Type       string                   `json:"type"`
	Role       string                   `json:"role"`
	Content    []AnthropicContentBlock  `json:"content"`
	Model      string                   `json:"model"`
	StopReason string                   `json:"stop_reason"`
	Usage      AnthropicUsage           `json:"usage"`
}

// AnthropicUsage 表示 Anthropic token 用量
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// AnthropicError 表示 Anthropic 错误响应
type AnthropicError struct {
	Type  string              `json:"type"` // always "error"
	Error AnthropicErrorBody  `json:"error"`
}

// AnthropicErrorBody 表示错误详情
type AnthropicErrorBody struct {
	Type    string `json:"type"`    // invalid_request_error, authentication_error, etc.
	Message string `json:"message"`
}
