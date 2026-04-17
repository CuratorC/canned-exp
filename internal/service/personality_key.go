package service

import (
	"context"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"
)

const (
	DefaultPKPage     = 1
	DefaultPKPageSize = 20
)

// PersonalityKeyService PersonalityKey 业务逻辑层
type PersonalityKeyService struct {
	repo *repository.PersonalityKeyGormRepo
}

// NewPersonalityKeyService 创建 PersonalityKeyService
func NewPersonalityKeyService(repo *repository.PersonalityKeyGormRepo) *PersonalityKeyService {
	return &PersonalityKeyService{repo: repo}
}

func (s *PersonalityKeyService) Save(ctx context.Context, pk *model.PersonalityKey) (uint, error) {
	if err := model.GetValidator().Struct(pk); err != nil {
		return 0, err
	}
	return s.repo.Save(ctx, pk)
}

func (s *PersonalityKeyService) Get(ctx context.Context, id uint) (*model.PersonalityKey, error) {
	return s.repo.Get(ctx, id)
}

func (s *PersonalityKeyService) GetByName(ctx context.Context, keyName string) (*model.PersonalityKey, error) {
	return s.repo.GetByName(ctx, keyName)
}

func (s *PersonalityKeyService) Update(ctx context.Context, pk *model.PersonalityKey) error {
	if err := model.GetValidator().Struct(pk); err != nil {
		return err
	}
	return s.repo.Update(ctx, pk)
}

func (s *PersonalityKeyService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *PersonalityKeyService) List(ctx context.Context, page, pageSize int) ([]model.PersonalityKey, int, error) {
	if page <= 0 {
		page = DefaultPKPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPKPageSize
	}
	return s.repo.List(ctx, page, pageSize)
}
