package proxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ConvertRequest 将 Anthropic Messages API 请求转换为 OpenAI Chat Completions API 请求。
// targetModel 是经过映射后的后端模型名。
// isStream 指示是否强制要求流式（当 Claude Code 通过 Accept header 声明流式时使用）。
func ConvertRequest(req *AnthropicRequest, targetModel string, isStream bool) (*OpenAIRequest, error) {
	var messages []OpenAIMessage

	// 处理 system 字段
	if req.System != nil {
		sysMsg, err := convertSystem(req.System)
		if err != nil {
			return nil, fmt.Errorf("convert system: %w", err)
		}
		messages = append(messages, *sysMsg)
	}

	// 转换消息
	for _, msg := range req.Messages {
		converted, err := convertMessage(&msg)
		if err != nil {
			return nil, fmt.Errorf("convert message (role=%s): %w", msg.Role, err)
		}
		messages = append(messages, converted...)
	}

	result := &OpenAIRequest{
		Model:       targetModel,
		Messages:    messages,
		MaxTokens:   req.MaxTokens,
		Stream:      req.Stream || isStream,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	if len(req.StopSequences) > 0 {
		result.Stop = req.StopSequences
	}

	// 转换工具定义
	if len(req.Tools) > 0 {
		result.Tools = make([]OpenAITool, len(req.Tools))
		for i, tool := range req.Tools {
			result.Tools[i] = OpenAITool{
				Type: "function",
				Function: OpenAIFunction{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			}
		}
	}

	// 转换工具选择
	if req.ToolChoice != nil {
		result.ToolChoice = convertToolChoice(req.ToolChoice)
	}

	return result, nil
}

// convertSystem 将 Anthropic system 字段转换为 OpenAI system 消息
func convertSystem(raw json.RawMessage) (*OpenAIMessage, error) {
	// 尝试解析为字符串
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return &OpenAIMessage{Role: "system", Content: s}, nil
	}

	// 解析为内容块数组
	var blocks []AnthropicSystemBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("system must be string or array: %w", err)
	}

	var parts []string
	for _, b := range blocks {
		if b.Type == "text" {
			parts = append(parts, b.Text)
		}
	}
	return &OpenAIMessage{Role: "system", Content: strings.Join(parts, "")}, nil
}

// convertMessage 将一条 Anthropic 消息转换为一条或多条 OpenAI 消息
func convertMessage(msg *AnthropicMessage) ([]OpenAIMessage, error) {
	blocks, err := msg.ContentBlocks()
	if err != nil {
		return nil, err
	}

	switch msg.Role {
	case "user":
		return convertUserMessage(blocks)
	case "assistant":
		return convertAssistantMessage(blocks)
	default:
		// 其他 role 直接作为单条消息
		var textParts []string
		for _, b := range blocks {
			if b.Type == "text" {
				textParts = append(textParts, b.Text)
			}
		}
		return []OpenAIMessage{{
			Role:    msg.Role,
			Content: strings.Join(textParts, ""),
		}}, nil
	}
}

// convertUserMessage 转换用户消息，将 text 和 tool_result 拆分
func convertUserMessage(blocks []AnthropicContentBlock) ([]OpenAIMessage, error) {
	var messages []OpenAIMessage
	var textParts []string

	for _, block := range blocks {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_result":
			// 先将累积的 text 作为一个 user 消息
			if len(textParts) > 0 {
				messages = append(messages, OpenAIMessage{
					Role:    "user",
					Content: strings.Join(textParts, ""),
				})
				textParts = nil
			}
			// tool_result 转为 role:tool 消息
			content := fmt.Sprintf("%v", block.Content)
			messages = append(messages, OpenAIMessage{
				Role:       "tool",
				Content:    content,
				ToolCallID: block.ToolUseID,
			})
		}
	}

	// 剩余的 text 部分
	if len(textParts) > 0 {
		messages = append(messages, OpenAIMessage{
			Role:    "user",
			Content: strings.Join(textParts, ""),
		})
	}

	return messages, nil
}

// convertAssistantMessage 转换助手消息，提取 tool_use 为 tool_calls
func convertAssistantMessage(blocks []AnthropicContentBlock) ([]OpenAIMessage, error) {
	var textParts []string
	var toolCalls []OpenAIToolCall

	for _, block := range blocks {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   block.ID,
				Type: "function",
				Function: OpenAIFunction{
					Name:      block.Name,
					Arguments: string(block.Input),
				},
			})
		case "thinking":
			// thinking blocks 没有 OpenAI 对应，跳过
		}
	}

	msg := OpenAIMessage{
		Role:      "assistant",
		Content:   strings.Join(textParts, ""),
		ToolCalls: toolCalls,
	}
	if len(toolCalls) > 0 {
		// 确保有 ToolCalls 时清空空 content
		// OpenAI 要求有 tool_calls 时 content 可为空字符串
	}
	return []OpenAIMessage{msg}, nil
}

// convertToolChoice 转换工具选择策略
func convertToolChoice(tc *AnthropicToolChoice) interface{} {
	switch tc.Type {
	case "auto":
		return "auto"
	case "any":
		return "required"
	case "none":
		return "none"
	case "tool":
		return map[string]interface{}{
			"type": "function",
			"function": map[string]string{
				"name": tc.Name,
			},
		}
	default:
		return "auto"
	}
}

// ConvertResponse 将 OpenAI Chat Completions 响应转换为 Anthropic Messages API 响应
func ConvertResponse(resp *OpenAIResponse, originalModel, msgID string) *AnthropicResponse {
	result := &AnthropicResponse{
		ID:         msgID,
		Type:       "message",
		Role:       "assistant",
		Model:      originalModel,
		StopReason: "end_turn",
		Usage: AnthropicUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	if len(resp.Choices) == 0 {
		return result
	}

	choice := resp.Choices[0]
	result.StopReason = mapFinishReason(choice.FinishReason)

	// 转换内容
	var content []AnthropicContentBlock

	if choice.Message.Content != "" {
		content = append(content, AnthropicContentBlock{
			Type: "text",
			Text: choice.Message.Content,
		})
	}

	for _, tc := range choice.Message.ToolCalls {
		content = append(content, AnthropicContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	result.Content = content
	return result
}

// mapFinishReason 将 OpenAI finish_reason 映射为 Anthropic stop_reason
func mapFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "end_turn"
	case "tool_calls":
		return "tool_use"
	case "length":
		return "max_tokens"
	default:
		return "end_turn"
	}
}

// ConvertError 将 OpenAI 错误响应转换为 Anthropic 错误格式
func ConvertError(body []byte, statusCode int) *AnthropicError {
	errType := "api_error"
	msg := "upstream provider error"

	switch {
	case statusCode >= 400 && statusCode < 500:
		switch statusCode {
		case 400:
			errType = "invalid_request_error"
		case 401:
			errType = "authentication_error"
		case 403:
			errType = "permission_error"
		case 404:
			errType = "not_found_error"
		case 429:
			errType = "rate_limit_error"
		default:
			errType = "invalid_request_error"
		}
	}

	// 尝试从 body 提取错误消息
	if body != nil {
		var errBody OpenAIErrorBody
		if err := json.Unmarshal(body, &errBody); err == nil && errBody.Error.Message != "" {
			msg = errBody.Error.Message
		}
	}

	return &AnthropicError{
		Type: "error",
		Error: AnthropicErrorBody{
			Type:    errType,
			Message: msg,
		},
	}
}
