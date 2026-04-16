package vectorstore

import (
	"context"
	"database/sql"
	"encoding/binary"
	"math"
	"sort"
)

// SearchResult 向量搜索结果
type SearchResult struct {
	ID    string
	Score float64
}

// VectorStore 定义了向量存储和检索的抽象接口
type VectorStore interface {
	// Store 存储一个向量，关联到指定 ID（覆盖更新）
	Store(ctx context.Context, id string, vector []float64) error
	// Search 搜索与 query 最相似的 topK 个向量
	Search(ctx context.Context, query []float64, topK int) ([]SearchResult, error)
	// Delete 删除指定 ID 的向量（不存在时不报错）
	Delete(ctx context.Context, id string) error
}

// SQLiteVecStore 基于 SQLite BLOB 的向量存储实现
// 使用纯 Go 余弦相似度计算，适合个人经验库的规模
type SQLiteVecStore struct {
	db *sql.DB
}

// NewSQLiteVecStore 创建并初始化向量存储
func NewSQLiteVecStore(db *sql.DB) (*SQLiteVecStore, error) {
	s := &SQLiteVecStore{db: db}
	if err := s.init(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteVecStore) init() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS vectors (
			id TEXT PRIMARY KEY,
			vector BLOB NOT NULL
		)
	`)
	return err
}

func (s *SQLiteVecStore) Store(ctx context.Context, id string, vector []float64) error {
	data := encodeVector(vector)
	_, err := s.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO vectors (id, vector) VALUES (?, ?)",
		id, data,
	)
	return err
}

func (s *SQLiteVecStore) Search(ctx context.Context, query []float64, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, "SELECT id, vector FROM vectors")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		vec := decodeVector(data)
		score := cosineSimilarity(query, vec)
		results = append(results, SearchResult{ID: id, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK < len(results) {
		results = results[:topK]
	}

	return results, nil
}

func (s *SQLiteVecStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM vectors WHERE id = ?", id)
	return err
}

// --- 向量编解码 ---

func encodeVector(vec []float64) []byte {
	buf := make([]byte, len(vec)*8)
	for i, v := range vec {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(v))
	}
	return buf
}

func decodeVector(data []byte) []float64 {
	n := len(data) / 8
	vec := make([]float64, n)
	for i := range vec {
		vec[i] = math.Float64frombits(binary.LittleEndian.Uint64(data[i*8:]))
	}
	return vec
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
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
