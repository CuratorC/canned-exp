package experience

import "context"

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
	SimilarExperiences []SearchResult
}

// Service 经验库业务逻辑层
type Service struct {
	repo Repository
}

// NewService 创建经验库服务
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Save(ctx context.Context, exp *Experience, force bool) (*SaveResult, error) {
	if err := exp.Validate(); err != nil {
		return nil, err
	}

	if !force {
		similar, err := s.repo.Search(ctx, exp.TextToEmbed(), 5)
		if err != nil {
			return nil, err
		}
		var duplicates []SearchResult
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

func (s *Service) Get(ctx context.Context, id string) (*Experience, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, exp *Experience) error {
	if err := exp.Validate(); err != nil {
		return err
	}
	return s.repo.Update(ctx, exp)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Search(ctx context.Context, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = DefaultTopK
	}
	if topK > MaxTopK {
		topK = MaxTopK
	}
	return s.repo.Search(ctx, query, topK)
}

func (s *Service) List(ctx context.Context, page, pageSize int) ([]Experience, int, error) {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	return s.repo.List(ctx, page, pageSize)
}
