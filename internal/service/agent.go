package service

import (
	"context"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"
)

const (
	DefaultAgentPage     = 1
	DefaultAgentPageSize = 20
)

// AgentService Agent 业务逻辑层
type AgentService struct {
	repo *repository.AgentGormRepo
}

// NewAgentService 创建 AgentService
func NewAgentService(repo *repository.AgentGormRepo) *AgentService {
	return &AgentService{repo: repo}
}

// Save 验证并保存 Agent
func (s *AgentService) Save(ctx context.Context, agent *model.Agent) (uint, error) {
	if err := model.GetValidator().Struct(agent); err != nil {
		return 0, err
	}
	return s.repo.Save(ctx, agent)
}

// Get 按 ID 获取 Agent
func (s *AgentService) Get(ctx context.Context, id uint) (*model.Agent, error) {
	return s.repo.Get(ctx, id)
}

// Update 验证并更新 Agent
func (s *AgentService) Update(ctx context.Context, agent *model.Agent) error {
	if err := model.GetValidator().Struct(agent); err != nil {
		return err
	}
	return s.repo.Update(ctx, agent)
}

// Delete 删除 Agent
func (s *AgentService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

// List 分页列出 Agent
func (s *AgentService) List(ctx context.Context, page, pageSize int) ([]model.Agent, int, error) {
	if page <= 0 {
		page = DefaultAgentPage
	}
	if pageSize <= 0 {
		pageSize = DefaultAgentPageSize
	}
	return s.repo.List(ctx, page, pageSize)
}
