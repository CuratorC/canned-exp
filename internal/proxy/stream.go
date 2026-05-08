package proxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// streamState 跟踪流式转换的状态
type streamState struct {
	contentBlockIndex int
	textBlockStarted  bool
	inToolUse         bool
	toolCallIndexMap  map[int]int // OpenAI tool_call index -> Anthropic content block index
	outputTokens      int
	messageStarted    bool
}

// StreamProxy 从 OpenAI SSE 响应读取 chunks，实时转换为 Anthropic SSE 事件输出
func StreamProxy(c *gin.Context, resp *http.Response, msgID, model string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		// 降级为非流式
		c.JSON(500, AnthropicError{
			Type: "error",
			Error: AnthropicErrorBody{Type: "api_error", Message: "streaming not supported"},
		})
		return
	}

	state := &streamState{
		toolCallIndexMap: make(map[int]int),
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}

		// 只处理 data: 开头的行
		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]
			if data == "[DONE]" {
				break
			}

			var chunk OpenAIChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			processChunk(c.Writer, flusher, &choice, state, msgID, model)
		}
	}

	// 确保所有 block 都关闭
	closeCurrentBlock(c.Writer, flusher, state)

	// 如果流意外结束但还没发 message_stop
	if state.messageStarted {
		emitSSE(c.Writer, flusher, "message_delta", map[string]interface{}{
			"delta": map[string]interface{}{"stop_reason": "end_turn"},
			"usage": map[string]interface{}{"output_tokens": state.outputTokens},
		})
		emitSSE(c.Writer, flusher, "message_stop", map[string]interface{}{})
	}
}

// processChunk 处理单个 OpenAI chunk
func processChunk(w io.Writer, flusher http.Flusher, choice *OpenAIChunkChoice, state *streamState, msgID, model string) {
	delta := choice.Delta

	// 首次收到数据：发送 message_start
	if !state.messageStarted {
		emitSSE(w, flusher, "message_start", map[string]interface{}{
			"type": "message_start",
			"message": map[string]interface{}{
				"id":      msgID,
				"type":    "message",
				"role":    "assistant",
				"content": []interface{}{},
				"model":   model,
				"usage":   map[string]interface{}{"input_tokens": 0, "output_tokens": 0},
			},
		})
		state.messageStarted = true
	}

	// 处理 role 宣告（OpenAI 首个 chunk 通常是 delta.role="assistant"）
	if delta.Role == "assistant" && delta.Content == "" && len(delta.ToolCalls) == 0 {
		return
	}

	// 处理文本内容
	if delta.Content != "" {
		if !state.textBlockStarted {
			emitSSE(w, flusher, "content_block_start", map[string]interface{}{
				"type":  "content_block_start",
				"index": state.contentBlockIndex,
				"content_block": map[string]interface{}{
					"type": "text",
					"text": "",
				},
			})
			state.textBlockStarted = true
		}
		emitSSE(w, flusher, "content_block_delta", map[string]interface{}{
			"type":  "content_block_delta",
			"index": state.contentBlockIndex,
			"delta": map[string]interface{}{
				"type": "text_delta",
				"text": delta.Content,
			},
		})
		state.outputTokens++
	}

	// 处理工具调用
	for _, tc := range delta.ToolCalls {
		// 新的 tool call（有 ID 表示是第一个 chunk）
		if tc.ID != "" {
			// 关闭之前的 block
			closeCurrentBlock(w, flusher, state)

			// 分配 content block index
			state.toolCallIndexMap[tc.Index] = state.contentBlockIndex
			state.contentBlockIndex++

			funcName := tc.Function.Name
			emitSSE(w, flusher, "content_block_start", map[string]interface{}{
				"type":  "content_block_start",
				"index": state.toolCallIndexMap[tc.Index],
				"content_block": map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  funcName,
					"input": map[string]interface{}{},
				},
			})
			state.inToolUse = true
		}

		// tool call arguments 增量
		if tc.Function.Arguments != "" {
			blockIdx := state.toolCallIndexMap[tc.Index]
			emitSSE(w, flusher, "content_block_delta", map[string]interface{}{
				"type":  "content_block_delta",
				"index": blockIdx,
				"delta": map[string]interface{}{
					"type":         "input_json_delta",
					"partial_json": tc.Function.Arguments,
				},
			})
		}
	}

	// 处理 finish_reason
	if choice.FinishReason != nil && *choice.FinishReason != "" {
		closeCurrentBlock(w, flusher, state)

		stopReason := mapFinishReason(*choice.FinishReason)
		emitSSE(w, flusher, "message_delta", map[string]interface{}{
			"type": "message_delta",
			"delta": map[string]interface{}{
				"stop_reason": stopReason,
			},
			"usage": map[string]interface{}{
				"output_tokens": state.outputTokens,
			},
		})
		emitSSE(w, flusher, "message_stop", map[string]interface{}{})

		// 标记已完成，防止 StreamProxy 末尾重复发送
		state.messageStarted = false
	}
}

// closeCurrentBlock 关闭当前打开的 content block
func closeCurrentBlock(w io.Writer, flusher http.Flusher, state *streamState) {
	if state.textBlockStarted {
		emitSSE(w, flusher, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": state.contentBlockIndex,
		})
		state.contentBlockIndex++
		state.textBlockStarted = false
	}
	if state.inToolUse {
		// 找到最后一个 tool use 的 index
		lastIdx := 0
		for _, idx := range state.toolCallIndexMap {
			if idx > lastIdx {
				lastIdx = idx
			}
		}
		emitSSE(w, flusher, "content_block_stop", map[string]interface{}{
			"type":  "content_block_stop",
			"index": lastIdx,
		})
		state.inToolUse = false
	}
}

// emitSSE 写入一个 SSE 事件并 flush
func emitSSE(w io.Writer, flusher http.Flusher, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
	flusher.Flush()
}
