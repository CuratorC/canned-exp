package repository

import (
	"context"
	"errors"
	"fmt"

	"canned-exp/internal/model"

	"github.com/CuratorC/gocanned/cerr"
	"gorm.io/gorm"
)

var ErrMemoryNotFound = cerr.New("memory not found")

type MemoryGormRepo struct {
	db *gorm.DB
}

func NewMemoryGormRepo(db *gorm.DB) *MemoryGormRepo {
	return &MemoryGormRepo{db: db}
}

func (r *MemoryGormRepo) Save(ctx context.Context, m *model.Memory) (uint, error) {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return 0, cerr.Wrap(err, "insert memory")
	}
	return m.ID, nil
}

func (r *MemoryGormRepo) GetByID(ctx context.Context, id uint) (*model.Memory, error) {
	var m model.Memory
	err := r.db.WithContext(ctx).First(&m, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMemoryNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get memory %d", id))
	}
	return &m, nil
}

func (r *MemoryGormRepo) GetByAgentAndPath(ctx context.Context, agentID uint, path string) (*model.Memory, error) {
	var m model.Memory
	err := r.db.WithContext(ctx).First(&m, "agent_id = ? AND path = ?", agentID, path).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrMemoryNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get memory agent=%d path=%s", agentID, path))
	}
	return &m, nil
}

func (r *MemoryGormRepo) Upsert(ctx context.Context, m *model.Memory) (uint, error) {
	var existing model.Memory
	err := r.db.WithContext(ctx).
		Where("agent_id = ? AND path = ?", m.AgentID, m.Path).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
			return 0, cerr.Wrap(err, "insert memory")
		}
		return m.ID, nil
	}
	if err != nil {
		return 0, cerr.Wrap(err, "query memory for upsert")
	}

	existing.Title = m.Title
	existing.Content = m.Content
	existing.MemoryDate = m.MemoryDate
	if err := r.db.WithContext(ctx).Save(&existing).Error; err != nil {
		return 0, cerr.Wrap(err, "update memory in upsert")
	}
	m.ID = existing.ID
	return m.ID, nil
}

func (r *MemoryGormRepo) Update(ctx context.Context, m *model.Memory) error {
	var exists int64
	r.db.WithContext(ctx).Model(&model.Memory{}).Where("id = ?", m.ID).Count(&exists)
	if exists == 0 {
		return ErrMemoryNotFound
	}
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update memory %d", m.ID))
	}
	return nil
}

func (r *MemoryGormRepo) Delete(ctx context.Context, agentID uint, path string) error {
	result := r.db.WithContext(ctx).
		Where("agent_id = ? AND path = ?", agentID, path).
		Delete(&model.Memory{})
	if result.Error != nil {
		return cerr.Wrap(result.Error, "delete memory")
	}
	return nil
}

func (r *MemoryGormRepo) ListByAgent(ctx context.Context, agentID uint, pathPrefix string, page, pageSize int) ([]model.Memory, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Memory{}).Where("agent_id = ?", agentID)
	if pathPrefix != "" {
		db = db.Where("path LIKE ?", pathPrefix+"%")
	}

	var total int64
	db.Count(&total)

	offset := (page - 1) * pageSize
	var memories []model.Memory
	err := db.Order("path ASC").
		Limit(pageSize).
		Offset(offset).
		Find(&memories).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list memories")
	}
	return memories, int(total), nil
}

func (r *MemoryGormRepo) ListByDateRange(ctx context.Context, agentID uint, from, to string, page, pageSize int) ([]model.Memory, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Memory{}).Where("agent_id = ?", agentID)
	if from != "" {
		db = db.Where("memory_date >= ?", from)
	}
	if to != "" {
		db = db.Where("memory_date <= ?", to)
	}
	db = db.Where("memory_date != ''")

	var total int64
	db.Count(&total)

	offset := (page - 1) * pageSize
	var memories []model.Memory
	err := db.Order("memory_date DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&memories).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list memories by date")
	}
	return memories, int(total), nil
}

func (r *MemoryGormRepo) Search(ctx context.Context, agentID uint, query string, page, pageSize int) ([]model.Memory, int, error) {
	db := r.db.WithContext(ctx).Model(&model.Memory{}).
		Where("agent_id = ? AND (content LIKE ? OR title LIKE ?)", agentID, "%"+query+"%", "%"+query+"%")

	var total int64
	db.Count(&total)

	offset := (page - 1) * pageSize
	var memories []model.Memory
	err := db.Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&memories).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "search memories")
	}
	return memories, int(total), nil
}
