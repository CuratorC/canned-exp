package embedding

import (
	"context"
	"math"
	"os"
	"testing"
)

// --- Mock 实现，用于验证接口契约 ---

type mockProvider struct {
	dimensions int
}

func (m *mockProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i := range texts {
		results[i] = make([]float64, m.dimensions)
		for j := range results[i] {
			results[i][j] = float64(i)
		}
	}
	return results, nil
}

func (m *mockProvider) Dimensions() int {
	return m.dimensions
}

// --- 接口契约测试 ---

func TestEmbeddingProvider_Interface(t *testing.T) {
	t.Run("单条文本返回一个向量", func(t *testing.T) {
		provider := &mockProvider{dimensions: 4}
		results, err := provider.Embed(context.Background(), []string{"测试文本"})
		if err != nil {
			t.Fatalf("Embed() error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("期望 1 个结果，得到 %d", len(results))
		}
		if len(results[0]) != 4 {
			t.Errorf("向量维度 = %d, want 4", len(results[0]))
		}
	})

	t.Run("批量文本返回等量向量", func(t *testing.T) {
		provider := &mockProvider{dimensions: 4}
		inputs := []string{"第一段", "第二段", "第三段"}
		results, err := provider.Embed(context.Background(), inputs)
		if err != nil {
			t.Fatalf("Embed() error: %v", err)
		}
		if len(results) != len(inputs) {
			t.Errorf("结果数量 = %d, want %d", len(results), len(inputs))
		}
		for i, vec := range results {
			if vec[0] != float64(i) {
				t.Errorf("results[%d][0] = %f, want %f", i, vec[0], float64(i))
			}
		}
	})

	t.Run("Dimensions 返回模型维度", func(t *testing.T) {
		provider := &mockProvider{dimensions: 2048}
		if d := provider.Dimensions(); d != 2048 {
			t.Errorf("Dimensions() = %d, want 2048", d)
		}
	})

	t.Run("空输入返回空切片", func(t *testing.T) {
		provider := &mockProvider{dimensions: 4}
		results, err := provider.Embed(context.Background(), []string{})
		if err != nil {
			t.Fatalf("Embed() error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("期望空结果，得到 %d 条", len(results))
		}
	})
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// --- 智谱 Embedding 集成测试 ---
// 运行时需设置环境变量：ZHIPU_API_KEY
// go test -run TestZhiPuEmbedding_E2E -v

func TestZhiPuEmbedding_E2E(t *testing.T) {
	apiKey := os.Getenv("ZHIPU_API_KEY")
	if apiKey == "" {
		t.Skip("跳过：未设置 ZHIPU_API_KEY 环境变量")
	}

	provider := NewZhiPuProvider(apiKey, ZhiPuDefaultModel, ZhiPuDefaultDimensions)

	t.Run("单条文本生成向量", func(t *testing.T) {
		results, err := provider.Embed(context.Background(), []string{"Go 语言是一门编译型语言"})
		if err != nil {
			t.Fatalf("Embed() error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("期望 1 个结果，得到 %d", len(results))
		}
		if len(results[0]) != ZhiPuDefaultDimensions {
			t.Errorf("向量维度 = %d, want %d", len(results[0]), ZhiPuDefaultDimensions)
		}
	})

	t.Run("批量文本生成等量向量", func(t *testing.T) {
		inputs := []string{
			"Go 测试文件要和源文件同目录",
			"Python 测试文件通常放在 tests 目录",
		}
		results, err := provider.Embed(context.Background(), inputs)
		if err != nil {
			t.Fatalf("Embed() error: %v", err)
		}
		if len(results) != len(inputs) {
			t.Errorf("结果数量 = %d, want %d", len(results), len(inputs))
		}
	})

	t.Run("语义相近的文本向量距离更近", func(t *testing.T) {
		inputs := []string{
			"Go 语言如何编写单元测试",
			"Go 测试的最佳实践",
			"今天北京天气怎么样",
		}
		results, err := provider.Embed(context.Background(), inputs)
		if err != nil {
			t.Fatalf("Embed() error: %v", err)
		}

		sim01 := cosineSimilarity(results[0], results[1])
		sim02 := cosineSimilarity(results[0], results[2])

		if sim01 <= sim02 {
			t.Errorf("相似文本的相似度 (%.4f) 应大于不相关文本 (%.4f)", sim01, sim02)
		}
	})
}
