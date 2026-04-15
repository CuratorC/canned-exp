package experience

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"canned-exp/internal/experience/embedding"
	"canned-exp/internal/experience/vectorstore"

	"github.com/google/uuid"
)

// ErrNotFound 经验不存在
var ErrNotFound = fmt.Errorf("experience not found")

// SearchResult 经验搜索结果
type SearchResult struct {
	Experience Experience
	Score      float64
}

// Repository 定义了经验持久化的抽象接口
type Repository interface {
	Save(ctx context.Context, exp *Experience) (string, error)
	Get(ctx context.Context, id string) (*Experience, error)
	Update(ctx context.Context, exp *Experience) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
	List(ctx context.Context, page, pageSize int) ([]Experience, int, error)
}

// SQLiteRepo 基于 SQLite 的经验仓库实现
type SQLiteRepo struct {
	db       *sql.DB
	vecStore *vectorstore.SQLiteVecStore
	embedder embedding.Provider
}

// NewSQLiteRepo 创建并初始化经验仓库
func NewSQLiteRepo(db *sql.DB, vecStore *vectorstore.SQLiteVecStore, embedder embedding.Provider) (*SQLiteRepo, error) {
	repo := &SQLiteRepo{db: db, vecStore: vecStore, embedder: embedder}
	if err := repo.init(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepo) init() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS experiences (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			tags TEXT NOT NULL DEFAULT '[]',
			source TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)
	`)
	return err
}

func (r *SQLiteRepo) Save(ctx context.Context, exp *Experience) (string, error) {
	id := uuid.New().String()
	now := time.Now().Format(time.RFC3339)

	// 生成向量
	text := exp.TextToEmbed()
	embeddings, err := r.embedder.Embed(ctx, []string{text})
	if err != nil {
		return "", fmt.Errorf("generate embedding: %w", err)
	}

	// 存储经验
	tagsJSON, _ := json.Marshal(exp.Tags)
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO experiences (id, title, content, tags, source, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, exp.Title, exp.Content, string(tagsJSON), exp.Source, now, now,
	)
	if err != nil {
		return "", fmt.Errorf("insert experience: %w", err)
	}

	// 存储向量
	if len(embeddings) > 0 {
		if err := r.vecStore.Store(ctx, id, embeddings[0]); err != nil {
			return "", fmt.Errorf("store vector: %w", err)
		}
	}

	exp.ID = id
	exp.CreatedAt = now
	exp.UpdatedAt = now

	return id, nil
}

func (r *SQLiteRepo) Get(ctx context.Context, id string) (*Experience, error) {
	var exp Experience
	var tagsJSON string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, content, tags, source, created_at, updated_at
		 FROM experiences WHERE id = ?`, id,
	).Scan(&exp.ID, &exp.Title, &exp.Content, &tagsJSON, &exp.Source, &exp.CreatedAt, &exp.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(tagsJSON), &exp.Tags)
	if exp.Tags == nil {
		exp.Tags = []string{}
	}
	return &exp, nil
}

func (r *SQLiteRepo) Update(ctx context.Context, exp *Experience) error {
	now := time.Now().Format(time.RFC3339)

	// 重新生成向量
	text := exp.TextToEmbed()
	embeddings, err := r.embedder.Embed(ctx, []string{text})
	if err != nil {
		return fmt.Errorf("generate embedding: %w", err)
	}

	tagsJSON, _ := json.Marshal(exp.Tags)
	result, err := r.db.ExecContext(ctx,
		`UPDATE experiences SET title = ?, content = ?, tags = ?, source = ?, updated_at = ?
		 WHERE id = ?`,
		exp.Title, exp.Content, string(tagsJSON), exp.Source, now, exp.ID,
	)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}

	// 更新向量
	if len(embeddings) > 0 {
		if err := r.vecStore.Store(ctx, exp.ID, embeddings[0]); err != nil {
			return fmt.Errorf("update vector: %w", err)
		}
	}

	exp.UpdatedAt = now
	return nil
}

func (r *SQLiteRepo) Delete(ctx context.Context, id string) error {
	r.db.ExecContext(ctx, "DELETE FROM experiences WHERE id = ?", id)
	r.vecStore.Delete(ctx, id)
	return nil
}

func (r *SQLiteRepo) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	// 生成查询向量
	embeddings, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("generate query embedding: %w", err)
	}
	if len(embeddings) == 0 {
		return nil, nil
	}

	// 向量搜索
	vecResults, err := r.vecStore.Search(ctx, embeddings[0], topK)
	if err != nil {
		return nil, err
	}

	// 加载完整经验
	results := make([]SearchResult, 0, len(vecResults))
	for _, vr := range vecResults {
		exp, err := r.Get(ctx, vr.ID)
		if err != nil {
			continue // 已删除的跳过
		}
		results = append(results, SearchResult{
			Experience: *exp,
			Score:      vr.Score,
		})
	}

	return results, nil
}

func (r *SQLiteRepo) List(ctx context.Context, page, pageSize int) ([]Experience, int, error) {
	var total int
	r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM experiences").Scan(&total)

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, content, tags, source, created_at, updated_at
		 FROM experiences ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		pageSize, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var experiences []Experience
	for rows.Next() {
		var exp Experience
		var tagsJSON string
		if err := rows.Scan(&exp.ID, &exp.Title, &exp.Content, &tagsJSON, &exp.Source, &exp.CreatedAt, &exp.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal([]byte(tagsJSON), &exp.Tags)
		if exp.Tags == nil {
			exp.Tags = []string{}
		}
		experiences = append(experiences, exp)
	}

	return experiences, total, nil
}
