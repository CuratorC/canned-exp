package vectorstore

import (
	"context"
	"database/sql"
	"math"
	"testing"

	_ "modernc.org/sqlite"
)

// helper: 生成一个指定维度的向量，各分量均匀分布
func makeVector(dimensions int, value float64) []float64 {
	vec := make([]float64, dimensions)
	for i := range vec {
		vec[i] = value
	}
	return vec
}

// helper: 两个完全相同方向的向量（余弦相似度 = 1）
func makeSameDirectionVectors(dimensions int) ([]float64, []float64) {
	a := make([]float64, dimensions)
	b := make([]float64, dimensions)
	for i := 0; i < dimensions; i++ {
		a[i] = float64(i + 1)
		b[i] = float64(i+1) * 2 // 同方向，不同模长
	}
	return a, b
}

// newTestStore 创建内存 SQLite 的 VectorStore 用于测试
func newTestStore(t *testing.T, dimensions int) VectorStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open memory sqlite: %v", err)
	}
	store, err := NewSQLiteVecStore(db)
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}
	return store
}

func TestVectorStore_CRUD(t *testing.T) {
	store := newTestStore(t, 4)
	ctx := context.Background()

	t.Run("存入后能按 ID 检索", func(t *testing.T) {
		vec := makeVector(4, 1.0)
		err := store.Store(ctx, "exp-1", vec)
		if err != nil {
			t.Fatalf("Store() error: %v", err)
		}

		results, err := store.Search(ctx, vec, 1)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("期望 1 条结果，得到 %d", len(results))
		}
		if results[0].ID != "exp-1" {
			t.Errorf("ID = %q, want %q", results[0].ID, "exp-1")
		}
	})

	t.Run("删除后搜索不到", func(t *testing.T) {
		vec := makeVector(4, 2.0)
		store.Store(ctx, "exp-to-delete", vec)

		err := store.Delete(ctx, "exp-to-delete")
		if err != nil {
			t.Fatalf("Delete() error: %v", err)
		}

		results, err := store.Search(ctx, vec, 10)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		for _, r := range results {
			if r.ID == "exp-to-delete" {
				t.Error("已删除的向量仍然出现在搜索结果中")
			}
		}
	})

	t.Run("相同 ID 覆盖更新", func(t *testing.T) {
		old := makeVector(4, 1.0)
		new := makeVector(4, 9.0)

		store.Store(ctx, "exp-overwrite", old)
		store.Store(ctx, "exp-overwrite", new)

		results, _ := store.Search(ctx, new, 10)
		found := false
		for _, r := range results {
			if r.ID == "exp-overwrite" {
				found = true
				break
			}
		}
		if !found {
			t.Error("覆盖更新后的向量未被搜索到")
		}
	})
}

func TestVectorStore_Search(t *testing.T) {
	store := newTestStore(t, 4)
	ctx := context.Background()

	a, b := makeSameDirectionVectors(4)
	c2 := make([]float64, 4)
	c2[0] = 0
	c2[1] = 1.0
	c2[2] = 0
	c2[3] = 0

	store.Store(ctx, "exp-a", a)
	store.Store(ctx, "exp-b", b)
	store.Store(ctx, "exp-c", c2)

	t.Run("搜索结果按相似度降序排列", func(t *testing.T) {
		results, err := store.Search(ctx, a, 3)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) < 2 {
			t.Fatalf("期望至少 2 条结果，得到 %d", len(results))
		}

		for i := 0; i < len(results)-1; i++ {
			if results[i].Score < results[i+1].Score {
				t.Errorf("结果未按相似度降序: score[%d]=%.4f < score[%d]=%.4f",
					i, results[i].Score, i+1, results[i+1].Score)
			}
		}
	})

	t.Run("同方向向量相似度高于正交向量", func(t *testing.T) {
		results, _ := store.Search(ctx, a, 3)

		var scoreB, scoreC float64
		for _, r := range results {
			switch r.ID {
			case "exp-b":
				scoreB = r.Score
			case "exp-c":
				scoreC = r.Score
			}
		}
		if scoreB <= scoreC {
			t.Errorf("同方向相似度 (%.4f) 应大于正交相似度 (%.4f)", scoreB, scoreC)
		}
	})

	t.Run("topK 限制返回数量", func(t *testing.T) {
		results, _ := store.Search(ctx, a, 2)
		if len(results) > 2 {
			t.Errorf("期望最多 2 条结果，得到 %d", len(results))
		}
	})

	t.Run("topK 大于总数时返回全部", func(t *testing.T) {
		results, _ := store.Search(ctx, a, 100)
		if len(results) != 3 {
			t.Errorf("期望 3 条结果，得到 %d", len(results))
		}
	})
}

func TestVectorStore_Empty(t *testing.T) {
	store := newTestStore(t, 4)
	ctx := context.Background()

	t.Run("空库搜索返回空结果", func(t *testing.T) {
		query := makeVector(4, 1.0)
		results, err := store.Search(ctx, query, 5)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("期望 0 条结果，得到 %d", len(results))
		}
	})

	t.Run("删除不存在的 ID 不报错", func(t *testing.T) {
		err := store.Delete(ctx, "nonexistent")
		if err != nil {
			t.Errorf("删除不存在的 ID 不应报错，得到: %v", err)
		}
	})
}

// cosineSimilarity 用于测试断言（未在测试中使用但保留作为工具）
var _ = math.Sqrt // 确保 math 可用
