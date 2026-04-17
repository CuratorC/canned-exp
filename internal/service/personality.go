package service

import (
	"context"
	"fmt"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"
)

const (
	DefaultPersonalityPage     = 1
	DefaultPersonalityPageSize = 20
)

// PersonalityService Personality 业务逻辑层
type PersonalityService struct {
	repo     *repository.PersonalityGormRepo
	keyRepo  *repository.PersonalityKeyGormRepo
}

// NewPersonalityService 创建 PersonalityService
func NewPersonalityService(repo *repository.PersonalityGormRepo, keyRepo *repository.PersonalityKeyGormRepo) *PersonalityService {
	return &PersonalityService{repo: repo, keyRepo: keyRepo}
}

// Set 设置人格属性（Upsert 语义：不存在则创建，已存在则更新）
// 通过 key_name 查找 key_id，验证 key 存在后执行 upsert
func (s *PersonalityService) Set(ctx context.Context, p *model.Personality) (uint, error) {
	if err := model.GetValidator().Struct(p); err != nil {
		return 0, err
	}

	// 验证 key_id 存在
	if _, err := s.keyRepo.Get(ctx, p.KeyID); err != nil {
		return 0, fmt.Errorf("key_id %d does not exist: %w", p.KeyID, err)
	}

	return s.repo.Upsert(ctx, p)
}

func (s *PersonalityService) Get(ctx context.Context, id uint) (*model.Personality, error) {
	return s.repo.Get(ctx, id)
}

func (s *PersonalityService) Update(ctx context.Context, p *model.Personality) error {
	if err := model.GetValidator().Struct(p); err != nil {
		return err
	}
	return s.repo.Update(ctx, p)
}

func (s *PersonalityService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *PersonalityService) ListByAgent(ctx context.Context, agentID uint, page, pageSize int) ([]model.Personality, int, error) {
	if page <= 0 {
		page = DefaultPersonalityPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPersonalityPageSize
	}
	return s.repo.ListByAgent(ctx, agentID, page, pageSize)
}
