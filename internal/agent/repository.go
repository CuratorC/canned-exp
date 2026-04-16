package agent

import (
	"context"
	"errors"
	"fmt"

	"github.com/CuratorC/gocanned/cerr"
	"gorm.io/gorm"
)

// ErrNotFound 表示 Agent 不存在
var ErrNotFound = cerr.New("agent not found")

// Repository 定义 Agent 数据访问接口
type Repository interface {
	Save(ctx context.Context, agent *Agent) (string, error)
	Get(ctx context.Context, id string) (*Agent, error)
	Update(ctx context.Context, agent *Agent) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int) ([]Agent, int, error)
}

// GormRepo 基于 GORM 的 Agent Repository 实现
type GormRepo struct {
	db *gorm.DB
}

// NewGormRepo 创建 GormRepo
func NewGormRepo(db *gorm.DB) *GormRepo {
	return &GormRepo{db: db}
}

// Save 创建新 Agent，返回生成的 ID
func (r *GormRepo) Save(ctx context.Context, agent *Agent) (string, error) {
	if err := r.db.WithContext(ctx).Create(agent).Error; err != nil {
		return "", cerr.Wrap(err, "insert agent")
	}
	return agent.ID, nil
}

// Get 按 ID 获取 Agent
func (r *GormRepo) Get(ctx context.Context, id string) (*Agent, error) {
	var a Agent
	err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get agent %s", id))
	}
	return &a, nil
}

// Update 更新已有 Agent
func (r *GormRepo) Update(ctx context.Context, agent *Agent) error {
	// GORM Save 是 upsert，需先确认记录存在
	var exists int64
	r.db.WithContext(ctx).Model(&Agent{}).Where("id = ?", agent.ID).Count(&exists)
	if exists == 0 {
		return ErrNotFound
	}

	if err := r.db.WithContext(ctx).Save(agent).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update agent %s", agent.ID))
	}
	return nil
}

// Delete 删除 Agent
func (r *GormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Agent{}, "id = ?", id)
	if result.Error != nil {
		return cerr.Wrap(result.Error, fmt.Sprintf("delete agent %s", id))
	}
	return nil
}

// List 分页列出所有 Agent（按创建时间降序）
func (r *GormRepo) List(ctx context.Context, page, pageSize int) ([]Agent, int, error) {
	var total int64
	r.db.WithContext(ctx).Model(&Agent{}).Count(&total)

	offset := (page - 1) * pageSize
	var agents []Agent
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&agents).Error
	if err != nil {
		return nil, 0, cerr.Wrap(err, "list agents")
	}
	return agents, int(total), nil
}
