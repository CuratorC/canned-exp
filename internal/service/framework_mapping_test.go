package service

import (
	"context"
	"testing"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"

	_ "canned-exp/internal/database/migrations/main"

	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/migration"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMappingService(t *testing.T) *FrameworkMappingService {
	t.Helper()
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm: %v", err)
	}
	dbDB := &database.DB{Gorm: gormDB, Driver: "sqlite"}
	runner := migration.NewRunner("main", dbDB)
	if err := runner.Run(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	repo := repository.NewFrameworkMappingGormRepo(gormDB)
	keyRepo := repository.NewPersonalityKeyGormRepo(gormDB)
	return NewFrameworkMappingService(repo, keyRepo)
}

func TestFrameworkMappingService_Save(t *testing.T) {
	svc := setupMappingService(t)
	ctx := context.Background()

	t.Run("合法输入通过", func(t *testing.T) {
		id, err := svc.Save(ctx, &model.FrameworkMapping{
			KeyID:      1,
			Framework:  "custom",
			ConfigPath: "custom.md",
		})
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("无效 key_id 被拒绝", func(t *testing.T) {
		_, err := svc.Save(ctx, &model.FrameworkMapping{
			KeyID:      999,
			Framework:  "custom",
			ConfigPath: "x.md",
		})
		if err == nil {
			t.Error("expected error for non-existent key_id")
		}
	})

	t.Run("空 framework 被拒绝", func(t *testing.T) {
		_, err := svc.Save(ctx, &model.FrameworkMapping{
			KeyID:      1,
			Framework:  "",
			ConfigPath: "x.md",
		})
		if err == nil {
			t.Error("expected validation error")
		}
	})
}

func TestFrameworkMappingService_GetByKeyAndFramework(t *testing.T) {
	svc := setupMappingService(t)
	ctx := context.Background()

	t.Run("获取预置映射", func(t *testing.T) {
		m, err := svc.GetByKeyAndFramework(ctx, 1, "openclaw")
		if err != nil {
			t.Fatalf("GetByKeyAndFramework failed: %v", err)
		}
		if m.ConfigPath != "SOUL.md" {
			t.Errorf("expected 'SOUL.md', got %q", m.ConfigPath)
		}
	})
}

func TestFrameworkMappingService_ListByFramework(t *testing.T) {
	svc := setupMappingService(t)
	ctx := context.Background()

	t.Run("openclaw 有 6 条", func(t *testing.T) {
		mappings, total, err := svc.ListByFramework(ctx, "openclaw", 1, 10)
		if err != nil {
			t.Fatalf("ListByFramework failed: %v", err)
		}
		if total != 6 {
			t.Errorf("expected 6, got %d", total)
		}
		if len(mappings) != 6 {
			t.Errorf("expected 6 items, got %d", len(mappings))
		}
	})

	t.Run("默认分页", func(t *testing.T) {
		mappings, _, err := svc.ListByFramework(ctx, "openclaw", 0, 0)
		if err != nil {
			t.Fatalf("ListByFramework with defaults failed: %v", err)
		}
		if len(mappings) == 0 {
			t.Error("expected mappings")
		}
	})
}

func TestFrameworkMappingService_Update(t *testing.T) {
	svc := setupMappingService(t)
	ctx := context.Background()

	t.Run("更新 config_path", func(t *testing.T) {
		m, _ := svc.Get(ctx, 1)
		m.ConfigPath = "updated.md"
		if err := svc.Update(ctx, m); err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		got, _ := svc.Get(ctx, 1)
		if got.ConfigPath != "updated.md" {
			t.Errorf("expected 'updated.md', got %q", got.ConfigPath)
		}
	})
}

func TestFrameworkMappingService_Delete(t *testing.T) {
	svc := setupMappingService(t)
	ctx := context.Background()

	t.Run("删除", func(t *testing.T) {
		if err := svc.Delete(ctx, 1); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := svc.Get(ctx, 1); err == nil {
			t.Error("expected error after delete")
		}
	})
}
