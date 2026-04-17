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

// ExperienceGormRepo 基于 GORM 的经验仓库实现
type ExperienceGormRepo struct {
	db       *gorm.DB
	vecStore *vectorstore.SQLiteVecStore
	embedder embedding.Provider
}

// NewExperienceGormRepo 创建经验仓库
func NewExperienceGormRepo(db *gorm.DB, vecStore *vectorstore.SQLiteVecStore, embedder embedding.Provider) *ExperienceGormRepo {
	return &ExperienceGormRepo{db: db, vecStore: vecStore, embedder: embedder}
}

func (r *ExperienceGormRepo) Save(ctx context.Context, exp *model.Experience) (string, error) {
	text := exp.TextToEmbed()
	embeddings, err := r.embedder.Embed(ctx, []string{text})
	if err != nil {
		return "", cerr.Wrap(err, "generate embedding")
	}

	if err := r.db.WithContext(ctx).Create(exp).Error; err != nil {
		return "", cerr.Wrap(err, "insert experience")
	}

	if len(embeddings) > 0 {
		if err := r.vecStore.Store(ctx, exp.ID, embeddings[0]); err != nil {
			return "", cerr.Wrap(err, "store vector")
		}
	}

	return exp.ID, nil
}

func (r *ExperienceGormRepo) Get(ctx context.Context, id string) (*model.Experience, error) {
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

func (r *ExperienceGormRepo) Update(ctx context.Context, exp *model.Experience) error {
	text := exp.TextToEmbed()
	embeddings, err := r.embedder.Embed(ctx, []string{text})
	if err != nil {
		return cerr.Wrap(err, "generate embedding")
	}

	var exists int64
	r.db.WithContext(ctx).Model(&model.Experience{}).Where("id = ?", exp.ID).Count(&exists)
	if exists == 0 {
		return ErrNotFound
	}

	if err := r.db.WithContext(ctx).Save(exp).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update experience %s", exp.ID))
	}

	if len(embeddings) > 0 {
		if err := r.vecStore.Store(ctx, exp.ID, embeddings[0]); err != nil {
			return cerr.Wrap(err, "update vector")
		}
	}

	return nil
}

func (r *ExperienceGormRepo) Delete(ctx context.Context, id string) error {
	r.db.WithContext(ctx).Delete(&model.Experience{}, "id = ?", id)
	r.vecStore.Delete(ctx, id)
	return nil
}

func (r *ExperienceGormRepo) Search(ctx context.Context, query string, agentID uint, topK int) ([]SearchResult, error) {
	embeddings, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, cerr.Wrap(err, "generate query embedding")
	}
	if len(embeddings) == 0 {
		return nil, nil
	}

	fetchSize := topK * 3
	vecResults, err := r.vecStore.Search(ctx, embeddings[0], fetchSize)
	if err != nil {
		return nil, cerr.Wrap(err, "vector search")
	}

	results := make([]SearchResult, 0, topK)
	for _, vr := range vecResults {
		exp, err := r.Get(ctx, vr.ID)
		if err != nil {
			continue
		}
		// agentID 非 0 时：只返回属于该 agent 或全局（agent_id=0）的经验
		if agentID != 0 && exp.AgentID != agentID && exp.AgentID != 0 {
			continue
		}
		results = append(results, SearchResult{
			Experience: *exp,
			Score:      vr.Score,
		})
		if len(results) >= topK {
			break
		}
	}

	return results, nil
}

func (r *ExperienceGormRepo) List(ctx context.Context, agentID uint, page, pageSize int) ([]model.Experience, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Experience{})
	if agentID != 0 {
		db = db.Where("agent_id = ? OR agent_id = 0", agentID)
	}

	var total int64
	db.Count(&total)

	var experiences []model.Experience
	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&experiences).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list experiences")
	}

	return experiences, int(total), nil
}
