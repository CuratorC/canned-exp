package repository

import (
	"context"
	"math"
	"testing"

	"canned-exp/internal/embedding"
	"canned-exp/internal/model"
	"canned-exp/internal/vectorstore"

	_ "canned-exp/internal/database/migrations/main"

	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/logger"
	"github.com/CuratorC/gocanned/migration"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"go.uber.org/zap"
)

func init() {
	logger.Logger = zap.NewNop()
}

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

// setupTestRepo 创建使用内存 SQLite + gocanned migration 的完整测试仓库
func setupTestRepo(t *testing.T) (*GormRepo, *gorm.DB) {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm memory sqlite: %v", err)
	}

	// 通过 gocanned migration 初始化表结构
	dbDB := &database.DB{Gorm: gormDB, Driver: "sqlite"}
	runner := migration.NewRunner("main", dbDB)
	if err := runner.Run(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	vecStore, err := vectorstore.NewSQLiteVecStore(dbDB.SQL())
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}

	embedder := &mockEmbeddingProvider{dimensions: 64}
	repo := NewGormRepo(gormDB, vecStore, embedder)
	return repo, gormDB
}

// 确保 mockEmbeddingProvider 实现了 embedding.Provider 接口
var _ embedding.Provider = (*mockEmbeddingProvider)(nil)

// 确保 model 包的 init() 被触发（注册 sonic serializer）
var _ = model.MaxContentLength
