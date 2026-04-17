package service

import (
	"context"
	"fmt"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"
)

const (
	DefaultMappingPage     = 1
	DefaultMappingPageSize = 20
)

// FrameworkMappingService FrameworkMapping 业务逻辑层
type FrameworkMappingService struct {
	repo    *repository.FrameworkMappingGormRepo
	keyRepo *repository.PersonalityKeyGormRepo
}

// NewFrameworkMappingService 创建 FrameworkMappingService
func NewFrameworkMappingService(repo *repository.FrameworkMappingGormRepo, keyRepo *repository.PersonalityKeyGormRepo) *FrameworkMappingService {
	return &FrameworkMappingService{repo: repo, keyRepo: keyRepo}
}

func (s *FrameworkMappingService) Save(ctx context.Context, m *model.FrameworkMapping) (uint, error) {
	if err := model.GetValidator().Struct(m); err != nil {
		return 0, err
	}
	if _, err := s.keyRepo.Get(ctx, m.KeyID); err != nil {
		return 0, fmt.Errorf("key_id %d does not exist: %w", m.KeyID, err)
	}
	return s.repo.Save(ctx, m)
}

func (s *FrameworkMappingService) Get(ctx context.Context, id uint) (*model.FrameworkMapping, error) {
	return s.repo.Get(ctx, id)
}

func (s *FrameworkMappingService) GetByKeyAndFramework(ctx context.Context, keyID uint, framework string) (*model.FrameworkMapping, error) {
	return s.repo.GetByKeyAndFramework(ctx, keyID, framework)
}

func (s *FrameworkMappingService) Update(ctx context.Context, m *model.FrameworkMapping) error {
	if err := model.GetValidator().Struct(m); err != nil {
		return err
	}
	return s.repo.Update(ctx, m)
}

func (s *FrameworkMappingService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *FrameworkMappingService) ListByFramework(ctx context.Context, framework string, page, pageSize int) ([]model.FrameworkMapping, int, error) {
	if page <= 0 {
		page = DefaultMappingPage
	}
	if pageSize <= 0 {
		pageSize = DefaultMappingPageSize
	}
	return s.repo.ListByFramework(ctx, framework, page, pageSize)
}

func (s *FrameworkMappingService) ListByKeyID(ctx context.Context, keyID uint) ([]model.FrameworkMapping, error) {
	return s.repo.ListByKeyID(ctx, keyID)
}
