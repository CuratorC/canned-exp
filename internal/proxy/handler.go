package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Config 代理配置
type Config struct {
	BaseURL      string
	APIKey       string
	ModelMap     string // JSON 编码的 map[string]string
	DefaultModel string // 未映射模型的回退模型名
}

// ProxyService 代理服务
type ProxyService struct {
	config       Config
	modelMap     map[string]string
	defaultModel string
	httpClient   *http.Client
}

// NewProxyService 创建代理服务实例
func NewProxyService(cfg Config) *ProxyService {
	modelMap := make(map[string]string)
	if cfg.ModelMap != "" {
		json.Unmarshal([]byte(cfg.ModelMap), &modelMap)
	}

	return &ProxyService{
		config:       cfg,
		modelMap:     modelMap,
		defaultModel: cfg.DefaultModel,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // 流式请求需要较长超时
		},
	}
}

// ModelName 通过模型映射表查找对应的模型名，未映射时回退到 DefaultModel
func (s *ProxyService) ModelName(anthropicModel string) string {
	if mapped, ok := s.modelMap[anthropicModel]; ok {
		return mapped
	}
	if s.defaultModel != "" {
		return s.defaultModel
	}
	return anthropicModel
}

// MessagesHandler 返回处理 /v1/messages 的 Gin handler
func MessagesHandler(svc *ProxyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 解析 Anthropic 请求
		var req AnthropicRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, AnthropicError{
				Type: "error",
				Error: AnthropicErrorBody{
					Type:    "invalid_request_error",
					Message: err.Error(),
				},
			})
			return
		}

		// 2. 判断是否流式（Claude Code 通过 Accept header 声明）
		accept := c.GetHeader("Accept")
		isStream := req.Stream || accept == "text/event-stream"

		// 3. 转换为 OpenAI 请求
		targetModel := svc.ModelName(req.Model)
		openaiReq, err := ConvertRequest(&req, targetModel, isStream)
		if err != nil {
			c.JSON(400, AnthropicError{
				Type: "error",
				Error: AnthropicErrorBody{
					Type:    "invalid_request_error",
					Message: err.Error(),
				},
			})
			return
		}

		// 3. 转发到后端
		body, _ := json.Marshal(openaiReq)
		httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", svc.config.BaseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			c.JSON(500, AnthropicError{
				Type:  "error",
				Error: AnthropicErrorBody{Type: "api_error", Message: "failed to create request"},
			})
			return
		}
		httpReq.Header.Set("Authorization", "Bearer "+svc.config.APIKey)
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := svc.httpClient.Do(httpReq)
		if err != nil {
			c.JSON(502, AnthropicError{
				Type:  "error",
				Error: AnthropicErrorBody{Type: "api_error", Message: "upstream provider unavailable"},
			})
			return
		}
		defer resp.Body.Close()

		// 4. 后端返回错误
		if resp.StatusCode != 200 {
			errBody, _ := io.ReadAll(resp.Body)
			fmt.Printf("[proxy] upstream error status=%d model=%s body=%s\n", resp.StatusCode, req.Model, string(errBody))
			c.JSON(resp.StatusCode, ConvertError(errBody, resp.StatusCode))
			return
		}

		// 5. 生成消息 ID
		msgID := "msg_" + uuid.New().String()[:24]

		// 6. 根据是否流式分发处理
		if isStream {
			StreamProxy(c, resp, msgID, req.Model)
		} else {
			respBody, _ := io.ReadAll(resp.Body)
			var openaiResp OpenAIResponse
			if err := json.Unmarshal(respBody, &openaiResp); err != nil {
				c.JSON(502, AnthropicError{
					Type:  "error",
					Error: AnthropicErrorBody{Type: "api_error", Message: "failed to parse upstream response"},
				})
				return
			}
			c.JSON(200, ConvertResponse(&openaiResp, req.Model, msgID))
		}
	}
}
