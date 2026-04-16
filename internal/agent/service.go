package agent

import "context"

const (
	DefaultAgentPage     = 1
	DefaultAgentPageSize = 20
)

// Service Agent 业务逻辑层
type Service struct {
	repo Repository
}

// NewService 创建 Agent Service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Save 验证并保存 Agent
func (s *Service) Save(ctx context.Context, agent *Agent) (string, error) {
	if err := agent.Validate(); err != nil {
		return "", err
	}
	return s.repo.Save(ctx, agent)
}

// Get 按 ID 获取 Agent
func (s *Service) Get(ctx context.Context, id string) (*Agent, error) {
	return s.repo.Get(ctx, id)
}

// Update 验证并更新 Agent
func (s *Service) Update(ctx context.Context, agent *Agent) error {
	if err := agent.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, agent)
}

// Delete 删除 Agent
func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// List 分页列出 Agent
func (s *Service) List(ctx context.Context, page, pageSize int) ([]Agent, int, error) {
	if page <= 0 {
		page = DefaultAgentPage
	}
	if pageSize <= 0 {
		pageSize = DefaultAgentPageSize
	}
	return s.repo.List(ctx, page, pageSize)
}
