package service

import (
	"context"
	"testing"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"

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

func setupPKService(t *testing.T) *PersonalityKeyService {
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
	repo := repository.NewPersonalityKeyGormRepo(gormDB)
	return NewPersonalityKeyService(repo)
}

func TestPersonalityKeyService_Save(t *testing.T) {
	svc := setupPKService(t)
	ctx := context.Background()

	t.Run("合法输入通过", func(t *testing.T) {
		id, err := svc.Save(ctx, &model.PersonalityKey{
			KeyName:     "custom_key",
			Description: "测试 key",
		})
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("空 key_name 被拒绝", func(t *testing.T) {
		_, err := svc.Save(ctx, &model.PersonalityKey{KeyName: ""})
		if err == nil {
			t.Error("expected error for empty key_name")
		}
	})
}

func TestPersonalityKeyService_Get(t *testing.T) {
	svc := setupPKService(t)
	ctx := context.Background()

	t.Run("获取预置 key", func(t *testing.T) {
		pk, err := svc.Get(ctx, 1)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if pk.KeyName != "system_prompt" {
			t.Errorf("expected 'system_prompt', got %q", pk.KeyName)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := svc.Get(ctx, 999)
		if err == nil {
			t.Error("expected error for non-existent key")
		}
	})
}

func TestPersonalityKeyService_GetByName(t *testing.T) {
	svc := setupPKService(t)
	ctx := context.Background()

	t.Run("按名称获取", func(t *testing.T) {
		pk, err := svc.GetByName(ctx, "emoji")
		if err != nil {
			t.Fatalf("GetByName failed: %v", err)
		}
		if pk.KeyName != "emoji" {
			t.Errorf("expected 'emoji', got %q", pk.KeyName)
		}
	})
}

func TestPersonalityKeyService_Update(t *testing.T) {
	svc := setupPKService(t)
	ctx := context.Background()

	t.Run("更新描述", func(t *testing.T) {
		pk, _ := svc.Get(ctx, 1)
		pk.Description = "新描述"
		if err := svc.Update(ctx, pk); err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		got, _ := svc.Get(ctx, 1)
		if got.Description != "新描述" {
			t.Errorf("expected '新描述', got %q", got.Description)
		}
	})

	t.Run("空 key_name 更新被拒绝", func(t *testing.T) {
		pk, _ := svc.Get(ctx, 1)
		pk.KeyName = ""
		if err := svc.Update(ctx, pk); err == nil {
			t.Error("expected validation error for empty key_name")
		}
	})
}

func TestPersonalityKeyService_Delete(t *testing.T) {
	svc := setupPKService(t)
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

func TestPersonalityKeyService_List(t *testing.T) {
	svc := setupPKService(t)
	ctx := context.Background()

	t.Run("列出所有预置 key", func(t *testing.T) {
		keys, total, err := svc.List(ctx, 1, 10)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 6 {
			t.Errorf("expected 6, got %d", total)
		}
		if len(keys) != 6 {
			t.Errorf("expected 6 keys, got %d", len(keys))
		}
	})

	t.Run("默认分页参数", func(t *testing.T) {
		keys, _, err := svc.List(ctx, 0, 0)
		if err != nil {
			t.Fatalf("List with defaults failed: %v", err)
		}
		if len(keys) == 0 {
			t.Error("expected keys with default pagination")
		}
	})
}
