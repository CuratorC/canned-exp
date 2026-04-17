package repository

import (
	"context"
	"errors"
	"fmt"

	"canned-exp/internal/model"

	"github.com/CuratorC/gocanned/cerr"
	"gorm.io/gorm"
)

// ErrPersonalityKeyNotFound key 不存在
var ErrPersonalityKeyNotFound = cerr.New("personality key not found")

// PersonalityKeyGormRepo 基于 GORM 的 PersonalityKey 数据访问
type PersonalityKeyGormRepo struct {
	db *gorm.DB
}

// NewPersonalityKeyGormRepo 创建 PersonalityKeyGormRepo
func NewPersonalityKeyGormRepo(db *gorm.DB) *PersonalityKeyGormRepo {
	return &PersonalityKeyGormRepo{db: db}
}

func (r *PersonalityKeyGormRepo) Save(ctx context.Context, pk *model.PersonalityKey) (uint, error) {
	if err := r.db.WithContext(ctx).Create(pk).Error; err != nil {
		return 0, cerr.Wrap(err, "insert personality key")
	}
	return pk.ID, nil
}

func (r *PersonalityKeyGormRepo) Get(ctx context.Context, id uint) (*model.PersonalityKey, error) {
	var pk model.PersonalityKey
	err := r.db.WithContext(ctx).First(&pk, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPersonalityKeyNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get personality key %d", id))
	}
	return &pk, nil
}

func (r *PersonalityKeyGormRepo) GetByName(ctx context.Context, keyName string) (*model.PersonalityKey, error) {
	var pk model.PersonalityKey
	err := r.db.WithContext(ctx).First(&pk, "key_name = ?", keyName).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrPersonalityKeyNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get personality key by name %s", keyName))
	}
	return &pk, nil
}

func (r *PersonalityKeyGormRepo) Update(ctx context.Context, pk *model.PersonalityKey) error {
	var exists int64
	r.db.WithContext(ctx).Model(&model.PersonalityKey{}).Where("id = ?", pk.ID).Count(&exists)
	if exists == 0 {
		return ErrPersonalityKeyNotFound
	}
	if err := r.db.WithContext(ctx).Save(pk).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update personality key %d", pk.ID))
	}
	return nil
}

func (r *PersonalityKeyGormRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.PersonalityKey{}, "id = ?", id)
	if result.Error != nil {
		return cerr.Wrap(result.Error, fmt.Sprintf("delete personality key %d", id))
	}
	return nil
}

func (r *PersonalityKeyGormRepo) List(ctx context.Context, page, pageSize int) ([]model.PersonalityKey, int, error) {
	var total int64
	r.db.WithContext(ctx).Model(&model.PersonalityKey{}).Count(&total)

	offset := (page - 1) * pageSize
	var keys []model.PersonalityKey
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&keys).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list personality keys")
	}
	return keys, int(total), nil
}
