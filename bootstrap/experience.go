package bootstrap

import (
	"canned-exp/internal/experience"
	"canned-exp/internal/experience/embedding"
	"canned-exp/internal/experience/vectorstore"
	"database/sql"
	"os"
	"path/filepath"

	"github.com/CuratorC/gocanned/cerr"
	"github.com/CuratorC/gocanned/config"
	_ "modernc.org/sqlite"
)

// SetupExperience 组装经验库的完整依赖链
func SetupExperience() (*experience.Service, error) {
	// 1. 数据库
	dbPath := config.GetString("experience.db_path")
	if dbPath == "" {
		dbPath = "storage/experience.db"
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, cerr.Wrapf(err, "create db directory %s", dir)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, cerr.Wrap(err, "open experience database")
	}
	// SQLite 单连接模式
	db.SetMaxOpenConns(1)

	// 2. Embedding Provider
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

	// 3. VectorStore
	vecStore, err := vectorstore.NewSQLiteVecStore(db)
	if err != nil {
		return nil, cerr.Wrap(err, "create vector store")
	}

	// 4. Repository
	repo, err := experience.NewSQLiteRepo(db, vecStore, embedder)
	if err != nil {
		return nil, cerr.Wrap(err, "create experience repository")
	}

	// 5. Service
	return experience.NewService(repo), nil
}
