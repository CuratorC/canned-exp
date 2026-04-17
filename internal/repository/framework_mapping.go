package repository

import (
	"context"
	"errors"
	"fmt"

	"canned-exp/internal/model"

	"github.com/CuratorC/gocanned/cerr"
	"gorm.io/gorm"
)

// ErrFrameworkMappingNotFound 映射不存在
var ErrFrameworkMappingNotFound = cerr.New("framework mapping not found")

// FrameworkMappingGormRepo 基于 GORM 的 FrameworkMapping 数据访问
type FrameworkMappingGormRepo struct {
	db *gorm.DB
}

// NewFrameworkMappingGormRepo 创建 FrameworkMappingGormRepo
func NewFrameworkMappingGormRepo(db *gorm.DB) *FrameworkMappingGormRepo {
	return &FrameworkMappingGormRepo{db: db}
}

func (r *FrameworkMappingGormRepo) Save(ctx context.Context, m *model.FrameworkMapping) (uint, error) {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return 0, cerr.Wrap(err, "insert framework mapping")
	}
	return m.ID, nil
}

func (r *FrameworkMappingGormRepo) Get(ctx context.Context, id uint) (*model.FrameworkMapping, error) {
	var m model.FrameworkMapping
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFrameworkMappingNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get framework mapping %d", id))
	}
	return &m, nil
}

func (r *FrameworkMappingGormRepo) GetByKeyAndFramework(ctx context.Context, keyID uint, framework string) (*model.FrameworkMapping, error) {
	var m model.FrameworkMapping
	err := r.db.WithContext(ctx).First(&m, "key_id = ? AND framework = ?", keyID, framework).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFrameworkMappingNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get mapping key=%d framework=%s", keyID, framework))
	}
	return &m, nil
}

func (r *FrameworkMappingGormRepo) Update(ctx context.Context, m *model.FrameworkMapping) error {
	var exists int64
	r.db.WithContext(ctx).Model(&model.FrameworkMapping{}).Where("id = ?", m.ID).Count(&exists)
	if exists == 0 {
		return ErrFrameworkMappingNotFound
	}
	if err := r.db.WithContext(ctx).Save(m).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update framework mapping %d", m.ID))
	}
	return nil
}

func (r *FrameworkMappingGormRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.FrameworkMapping{}, "id = ?", id)
	if result.Error != nil {
		return cerr.Wrap(result.Error, fmt.Sprintf("delete framework mapping %d", id))
	}
	return nil
}

func (r *FrameworkMappingGormRepo) ListByFramework(ctx context.Context, framework string, page, pageSize int) ([]model.FrameworkMapping, int, error) {
	db := r.db.WithContext(ctx).Model(&model.FrameworkMapping{}).Where("framework = ?", framework)

	var total int64
	db.Count(&total)

	offset := (page - 1) * pageSize
	var mappings []model.FrameworkMapping
	err := db.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&mappings).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list framework mappings")
	}
	return mappings, int(total), nil
}

func (r *FrameworkMappingGormRepo) ListByKeyID(ctx context.Context, keyID uint) ([]model.FrameworkMapping, error) {
	var mappings []model.FrameworkMapping
	err := r.db.WithContext(ctx).Where("key_id = ?", keyID).Find(&mappings).Error
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("list mappings for key %d", keyID))
	}
	return mappings, nil
}
