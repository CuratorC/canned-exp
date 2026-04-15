package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	ZhiPuDefaultModel      = "embedding-3"
	ZhiPuDefaultDimensions = 2048
	zhiPuBaseURL           = "https://open.bigmodel.cn/api/paas/v4/embeddings"
)

// Provider 定义了 Embedding 生成的抽象接口
type Provider interface {
	// Embed 将一组文本转化为向量
	Embed(ctx context.Context, texts []string) ([][]float64, error)
	// Dimensions 返回模型的输出向量维度
	Dimensions() int
}

// ZhiPuProvider 智谱 Embedding API 实现
type ZhiPuProvider struct {
	apiKey     string
	model      string
	dimensions int
	baseURL    string
	httpClient *http.Client
}

// NewZhiPuProvider 创建智谱 Embedding 提供者
func NewZhiPuProvider(apiKey, model string, dimensions int) *ZhiPuProvider {
	return &ZhiPuProvider{
		apiKey:     apiKey,
		model:      model,
		dimensions: dimensions,
		baseURL:    zhiPuBaseURL,
		httpClient: &http.Client{},
	}
}

// WithBaseURL 设置自定义 API 端点
func (p *ZhiPuProvider) WithBaseURL(url string) *ZhiPuProvider {
	if url != "" {
		p.baseURL = url
	}
	return p
}

type zhiPuRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type zhiPuResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
		Object    string    `json:"object"`
	} `json:"data"`
	Object string `json:"object"`
	Usage  struct {
		CompletionTokens int `json:"completion_tokens"`
		PromptTokens     int `json:"prompt_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *ZhiPuProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	reqBody := zhiPuRequest{
		Model:      p.model,
		Input:      texts,
		Dimensions: p.dimensions,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result zhiPuResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("zhipu api error: %s - %s", result.Error.Code, result.Error.Message)
	}

	embeddings := make([][]float64, len(result.Data))
	for _, d := range result.Data {
		embeddings[d.Index] = d.Embedding
	}

	return embeddings, nil
}

func (p *ZhiPuProvider) Dimensions() int {
	return p.dimensions
}
