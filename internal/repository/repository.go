package repository

import (
	"context"
	"errors"
	"fmt"

	"canned-exp/internal/embedding"
	"canned-exp/internal/model"
	"canned-exp/internal/vectorstore"

	"github.com/CuratorC/gocanned/cerr"
	"gorm.io/gorm"
)

// ErrNotFound 经验不存在
var ErrNotFound = cerr.New("experience not found")

// SearchResult 经验搜索结果
type SearchResult struct {
	Experience model.Experience
	Score      float64
}

// Repository 定义了经验持久化的抽象接口
type Repository interface {
	Save(ctx context.Context, exp *model.Experience) (string, error)
	Get(ctx context.Context, id string) (*model.Experience, error)
	Update(ctx context.Context, exp *model.Experience) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string, topK int) ([]SearchResult, error)
	List(ctx context.Context, page, pageSize int) ([]model.Experience, int, error)
}

// GormRepo 基于 GORM 的经验仓库实现
type GormRepo struct {
	db       *gorm.DB
	vecStore *vectorstore.SQLiteVecStore
	embedder embedding.Provider
}

// NewGormRepo 创建经验仓库
func NewGormRepo(db *gorm.DB, vecStore *vectorstore.SQLiteVecStore, embedder embedding.Provider) *GormRepo {
	return &GormRepo{db: db, vecStore: vecStore, embedder: embedder}
}

func (r *GormRepo) Save(ctx context.Context, exp *model.Experience) (string, error) {
	// 生成向量
	text := exp.TextToEmbed()
	embeddings, err := r.embedder.Embed(ctx, []string{text})
	if err != nil {
		return "", cerr.Wrap(err, "generate embedding")
	}

	// GORM BeforeCreate hook 会设置 ID 和时间戳
	if err := r.db.WithContext(ctx).Create(exp).Error; err != nil {
		return "", cerr.Wrap(err, "insert experience")
	}

	// 存储向量
	if len(embeddings) > 0 {
		if err := r.vecStore.Store(ctx, exp.ID, embeddings[0]); err != nil {
			return "", cerr.Wrap(err, "store vector")
		}
	}

	return exp.ID, nil
}

func (r *GormRepo) Get(ctx context.Context, id string) (*model.Experience, error) {
	var exp model.Experience
	err := r.db.WithContext(ctx).First(&exp, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get experience %s", id))
	}
	return &exp, nil
}

func (r *GormRepo) Update(ctx context.Context, exp *model.Experience) error {
	// 重新生成向量
	text := exp.TextToEmbed()
	embeddings, err := r.embedder.Embed(ctx, []string{text})
	if err != nil {
		return cerr.Wrap(err, "generate embedding")
	}

	// GORM Save 是 upsert，需先确认记录存在
	var exists int64
	r.db.WithContext(ctx).Model(&model.Experience{}).Where("id = ?", exp.ID).Count(&exists)
	if exists == 0 {
		return ErrNotFound
	}

	if err := r.db.WithContext(ctx).Save(exp).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update experience %s", exp.ID))
	}

	// 更新向量
	if len(embeddings) > 0 {
		if err := r.vecStore.Store(ctx, exp.ID, embeddings[0]); err != nil {
			return cerr.Wrap(err, "update vector")
		}
	}

	return nil
}

func (r *GormRepo) Delete(ctx context.Context, id string) error {
	r.db.WithContext(ctx).Delete(&model.Experience{}, "id = ?", id)
	r.vecStore.Delete(ctx, id)
	return nil
}

func (r *GormRepo) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	// 生成查询向量
	embeddings, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, cerr.Wrap(err, "generate query embedding")
	}
	if len(embeddings) == 0 {
		return nil, nil
	}

	// 向量搜索
	vecResults, err := r.vecStore.Search(ctx, embeddings[0], topK)
	if err != nil {
		return nil, cerr.Wrap(err, "vector search")
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

func (r *GormRepo) List(ctx context.Context, page, pageSize int) ([]model.Experience, int, error) {
	var total int64
	r.db.WithContext(ctx).Model(&model.Experience{}).Count(&total)

	var experiences []model.Experience
	offset := (page - 1) * pageSize
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&experiences).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list experiences")
	}

	return experiences, int(total), nil
}
