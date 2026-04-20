package service

import (
	"context"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"
)

const (
	DefaultMemoryPage     = 1
	DefaultMemoryPageSize = 20
)

// MemoryService Memory 业务逻辑层
type MemoryService struct {
	repo *repository.MemoryGormRepo
}

// NewMemoryService 创建 MemoryService
func NewMemoryService(repo *repository.MemoryGormRepo) *MemoryService {
	return &MemoryService{repo: repo}
}

// Set 保存记忆（Upsert 语义：agent_id + path 已存在则更新）
func (s *MemoryService) Set(ctx context.Context, m *model.Memory) (uint, error) {
	if err := model.GetValidator().Struct(m); err != nil {
		return 0, err
	}
	return s.repo.Upsert(ctx, m)
}

func (s *MemoryService) Get(ctx context.Context, id uint) (*model.Memory, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *MemoryService) GetByPath(ctx context.Context, agentID uint, path string) (*model.Memory, error) {
	return s.repo.GetByAgentAndPath(ctx, agentID, path)
}

func (s *MemoryService) Update(ctx context.Context, m *model.Memory) error {
	if err := model.GetValidator().Struct(m); err != nil {
		return err
	}
	return s.repo.Update(ctx, m)
}

func (s *MemoryService) Delete(ctx context.Context, agentID uint, path string) error {
	return s.repo.Delete(ctx, agentID, path)
}

func (s *MemoryService) List(ctx context.Context, agentID uint, pathPrefix string, page, pageSize int) ([]model.Memory, int, error) {
	if page <= 0 {
		page = DefaultMemoryPage
	}
	if pageSize <= 0 {
		pageSize = DefaultMemoryPageSize
	}
	return s.repo.ListByAgent(ctx, agentID, pathPrefix, page, pageSize)
}

func (s *MemoryService) ListByDateRange(ctx context.Context, agentID uint, from, to string, page, pageSize int) ([]model.Memory, int, error) {
	if page <= 0 {
		page = DefaultMemoryPage
	}
	if pageSize <= 0 {
		pageSize = DefaultMemoryPageSize
	}
	return s.repo.ListByDateRange(ctx, agentID, from, to, page, pageSize)
}

func (s *MemoryService) Search(ctx context.Context, agentID uint, query string, page, pageSize int) ([]model.Memory, int, error) {
	if page <= 0 {
		page = DefaultMemoryPage
	}
	if pageSize <= 0 {
		pageSize = DefaultMemoryPageSize
	}
	return s.repo.Search(ctx, agentID, query, page, pageSize)
}
