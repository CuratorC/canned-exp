package repository

import (
	"context"
	"errors"
	"fmt"

	"canned-exp/internal/model"

	"github.com/CuratorC/gocanned/cerr"
	"gorm.io/gorm"
)

// ErrAgentNotFound 表示 Agent 不存在
var ErrAgentNotFound = cerr.New("agent not found")

// AgentGormRepo 基于 GORM 的 Agent 数据访问
type AgentGormRepo struct {
	db *gorm.DB
}

// NewAgentGormRepo 创建 AgentGormRepo
func NewAgentGormRepo(db *gorm.DB) *AgentGormRepo {
	return &AgentGormRepo{db: db}
}

// Save 创建新 Agent，返回生成的 ID
func (r *AgentGormRepo) Save(ctx context.Context, agent *model.Agent) (uint, error) {
	if err := r.db.WithContext(ctx).Create(agent).Error; err != nil {
		return 0, cerr.Wrap(err, "insert agent")
	}
	return agent.ID, nil
}

// Get 按 ID 获取 Agent
func (r *AgentGormRepo) Get(ctx context.Context, id uint) (*model.Agent, error) {
	var a model.Agent
	err := r.db.WithContext(ctx).First(&a, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAgentNotFound
	}
	if err != nil {
		return nil, cerr.Wrap(err, fmt.Sprintf("get agent %d", id))
	}
	return &a, nil
}

// Update 更新已有 Agent
func (r *AgentGormRepo) Update(ctx context.Context, agent *model.Agent) error {
	var exists int64
	r.db.WithContext(ctx).Model(&model.Agent{}).Where("id = ?", agent.ID).Count(&exists)
	if exists == 0 {
		return ErrAgentNotFound
	}

	if err := r.db.WithContext(ctx).Save(agent).Error; err != nil {
		return cerr.Wrap(err, fmt.Sprintf("update agent %d", agent.ID))
	}
	return nil
}

// Delete 删除 Agent
func (r *AgentGormRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&model.Agent{}, "id = ?", id)
	if result.Error != nil {
		return cerr.Wrap(result.Error, fmt.Sprintf("delete agent %d", id))
	}
	return nil
}

// List 分页列出所有 Agent（按创建时间降序）
func (r *AgentGormRepo) List(ctx context.Context, page, pageSize int) ([]model.Agent, int, error) {
	var total int64
	r.db.WithContext(ctx).Model(&model.Agent{}).Count(&total)

	offset := (page - 1) * pageSize
	var agents []model.Agent
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
