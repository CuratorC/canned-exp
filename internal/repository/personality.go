package repository

import (
	"context"
	"errors"
	"fmt"

	"canned-exp/internal/model"

	"github.com/CuratorC/gocanned/cerr"
	"gorm.io/gorm"
)

// ErrPersonalityNotFound personality 不存在
var ErrPersonalityNotFound = cerr.New("personality not found")

// PersonalityGormRepo 基于 GORM 的 Personality 数据访问
type PersonalityGormRepo struct {
	db *gorm.DB
}

// NewPersonalityGormRepo 创建 PersonalityGormRepo
func NewPersonalityGormRepo(db *gorm.DB) *PersonalityGormRepo {
	return &PersonalityGormRepo{db: db}
}

func (r *PersonalityGormRepo) Save(ctx context.Context, p *model.Personality) (uint, error) {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return 0, cerr.Wrap(err, "insert personality")
	}
	return p.ID, nil
}

func (r *PersonalityGormRepo) Get(ctx context.Context, id uint) (*model.Personality, error) {
	var p model.Personality
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPersonalityNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get personality %d", id))
	}
	return &p, nil
}

func (r *PersonalityGormRepo) GetByAgentAndKey(ctx context.Context, agentID, keyID uint) (*model.Personality, error) {
	var p model.Personality
	err := r.db.WithContext(ctx).First(&p, "agent_id = ? AND key_id = ?", agentID, keyID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPersonalityNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get personality agent=%d key=%d", agentID, keyID))
	}
	return &p, nil
}

func (r *PersonalityGormRepo) Upsert(ctx context.Context, p *model.Personality) (uint, error) {
	var existing model.Personality
	err := r.db.WithContext(ctx).
		Where("agent_id = ? AND key_id = ?", p.AgentID, p.KeyID).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
			return 0, cerr.Wrap(err, "insert personality")
		}
		return p.ID, nil
	}
	if err != nil {
		return 0, cerr.Wrap(err, "query personality for upsert")
	}

	existing.Value = p.Value
	existing.Type = p.Type
	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return 0, cerr.Wrap(err, "update personality in upsert")
	}
	p.ID = existing.ID
	return p.ID, nil
}

func (r *PersonalityGormRepo) Update(ctx context.Context, p *model.Personality) error {
	var exists int64
	r.db.WithContext(ctx).Model(&model.Personality{}).Where("id = ?", p.ID).Count(&exists)
	if exists == 0 {
		return ErrPersonalityNotFound
	}
	if err := r.db.WithContext(ctx).Save(p).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update personality %d", p.ID))
	}
	return nil
}

func (r *PersonalityGormRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Personality{}, "id = ?", id)
	if result.Error != nil {
		return cerr.Wrap(result.Error, fmt.Sprintf("delete personality %d", id))
	}
	return nil
}

func (r *PersonalityGormRepo) ListAllByAgent(ctx context.Context, agentID uint) ([]model.Personality, error) {
	var personalities []model.Personality
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Order("key_id ASC").
		Find(&personalities).Error
	if err != nil {
		return nil, cerr.Wrap(err, "list all personalities by agent")
	}
	return personalities, nil
}

func (r *PersonalityGormRepo) ListByAgent(ctx context.Context, agentID uint, page, pageSize int) ([]model.Personality, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Personality{}).Where("agent_id = ?", agentID)

	var total int64
	db.Count(&total)

	offset := (page - 1) * pageSize
	var personalities []model.Personality
	err := db.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&personalities).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list personalities")
	}
	return personalities, int(total), nil
}
