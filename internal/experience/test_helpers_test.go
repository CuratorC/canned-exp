package experience

import (
	"context"
	"database/sql"
	"math"
	"testing"

	"canned-exp/internal/experience/embedding"
	"canned-exp/internal/experience/vectorstore"

	_ "modernc.org/sqlite"
)

// mockEmbeddingProvider 基于字符频率的 mock embedding
// 相似文本会产生相似向量，使语义搜索在测试中可用
type mockEmbeddingProvider struct {
	dimensions int
}

func (m *mockEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, text := range texts {
		results[i] = textToVector(text, m.dimensions)
	}
	return results, nil
}

func (m *mockEmbeddingProvider) Dimensions() int {
	return m.dimensions
}

// textToVector 将文本转化为基于字符频率的向量
// 用于测试中的语义模拟，不用于生产
func textToVector(text string, dims int) []float64 {
	vec := make([]float64, dims)
	for _, r := range text {
		idx := int(uint32(r)) % dims
		vec[idx] += 1.0
	}
	// 归一化
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for j := range vec {
			vec[j] /= norm
		}
	}
	return vec
}

// setupTestRepo 创建使用内存 SQLite 和 mock embedding 的完整测试仓库
func setupTestRepo(t *testing.T) (*SQLiteRepo, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open memory sqlite: %v", err)
	}

	vecStore, err := vectorstore.NewSQLiteVecStore(db)
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}

	embedder := &mockEmbeddingProvider{dimensions: 64}

	repo, err := NewSQLiteRepo(db, vecStore, embedder)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	return repo, db
}

// 确保 mockEmbeddingProvider 实现了 embedding.Provider 接口
var _ embedding.Provider = (*mockEmbeddingProvider)(nil)
