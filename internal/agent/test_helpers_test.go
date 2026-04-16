package agent

import (
	"testing"

	_ "canned-exp/internal/database/migrations/main"
	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/logger"
	"github.com/CuratorC/gocanned/migration"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"go.uber.org/zap"
)

func init() {
	logger.Logger = zap.NewNop()
}

// setupTestRepo 创建内存 SQLite + gocanned migration 的测试仓库
func setupTestRepo(t *testing.T) (*GormRepo, *gorm.DB) {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm memory sqlite: %v", err)
	}

	// 通过 gocanned migration 初始化表结构
	dbDB := &database.DB{Gorm: gormDB, Driver: "sqlite"}
	runner := migration.NewRunner("main", dbDB)
	if err := runner.Run(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := NewGormRepo(gormDB)
	return repo, gormDB
}
