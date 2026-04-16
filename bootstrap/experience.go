package bootstrap

import (
	"canned-exp/internal/embedding"
	"canned-exp/internal/repository"
	"canned-exp/internal/service"
	"canned-exp/internal/vectorstore"

	"github.com/CuratorC/gocanned/cerr"
	"github.com/CuratorC/gocanned/config"
	"github.com/CuratorC/gocanned/database"
)

// SetupExperience 组装经验库的完整依赖链
func SetupExperience(db *database.DB) (*service.ExperienceService, error) {
	// Embedding Provider
	apiKey := config.GetString("embedding.api_key")
	baseURL := config.GetString("embedding.base_url")
	model := config.GetString("embedding.model")
	dimensions := config.GetInt("embedding.dimensions")

	var embedder embedding.Provider
	provider := config.GetString("embedding.provider")
	switch provider {
	case "zhipu":
		if apiKey == "" {
			return nil, cerr.New("embedding.api_key is required")
		}
		embedder = embedding.NewZhiPuProvider(apiKey, model, dimensions).WithBaseURL(baseURL)
	default:
		return nil, cerr.Errorf("unsupported embedding provider: %s", provider)
	}

	// VectorStore（使用 *sql.DB）
	vecStore, err := vectorstore.NewSQLiteVecStore(db.SQL())
	if err != nil {
		return nil, cerr.Wrap(err, "create vector store")
	}

	// Repository（db.Gorm 提供 GORM 连接）
	repo := repository.NewGormRepo(db.Gorm, vecStore, embedder)

	// Service
	return service.NewExperienceService(repo), nil
}
