package service

import (
	"context"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"
)

const (
	DefaultTopK      = 5
	MaxTopK          = 50
	DefaultPage      = 1
	DefaultPageSize  = 20
	SimilarThreshold = 0.95
)

// SaveResult 保存操作的返回结果
type SaveResult struct {
	ID                 string
	Saved              bool
	SimilarExperiences []repository.SearchResult
}

// ExperienceService 经验库业务逻辑层
type ExperienceService struct {
	repo *repository.ExperienceGormRepo
}

// NewExperienceService 创建经验库服务
func NewExperienceService(repo *repository.ExperienceGormRepo) *ExperienceService {
	return &ExperienceService{repo: repo}
}

func (s *ExperienceService) Save(ctx context.Context, exp *model.Experience, force bool) (*SaveResult, error) {
	if err := model.GetValidator().Struct(exp); err != nil {
		return nil, err
	}

	if !force {
		similar, err := s.repo.Search(ctx, exp.TextToEmbed(), exp.AgentID, 5)
		if err != nil {
			return nil, err
		}
		var duplicates []repository.SearchResult
		for _, r := range similar {
			if r.Score >= SimilarThreshold {
				duplicates = append(duplicates, r)
			}
		}
		if len(duplicates) > 0 {
			return &SaveResult{Saved: false, SimilarExperiences: duplicates}, nil
		}
	}

	id, err := s.repo.Save(ctx, exp)
	if err != nil {
		return nil, err
	}
	return &SaveResult{ID: id, Saved: true}, nil
}

func (s *ExperienceService) Get(ctx context.Context, id string) (*model.Experience, error) {
	return s.repo.Get(ctx, id)
}

func (s *ExperienceService) Update(ctx context.Context, exp *model.Experience) error {
	if err := model.GetValidator().Struct(exp); err != nil {
		return err
	}
	return s.repo.Update(ctx, exp)
}

func (s *ExperienceService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *ExperienceService) Search(ctx context.Context, query string, agentID uint, topK int) ([]repository.SearchResult, error) {
	if topK <= 0 {
		topK = DefaultTopK
	}
	if topK > MaxTopK {
		topK = MaxTopK
	}
	return s.repo.Search(ctx, query, agentID, topK)
}

func (s *ExperienceService) List(ctx context.Context, agentID uint, page, pageSize int) ([]model.Experience, int, error) {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	return s.repo.List(ctx, agentID, page, pageSize)
}
