package repository

import (
	"context"
	"testing"

	"canned-exp/internal/model"
)

func TestFrameworkMappingRepo_Save(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFrameworkMappingGormRepo(db)
	ctx := context.Background()

	t.Run("正常保存并返回 ID", func(t *testing.T) {
		id, err := repo.Save(ctx, &model.FrameworkMapping{
			KeyID:      1,
			Framework:  "test-framework",
			ConfigPath: "test.md",
		})
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复 key_id+framework 失败", func(t *testing.T) {
		// 预置 openclaw + key_id=1 已存在
		_, err := repo.Save(ctx, &model.FrameworkMapping{
			KeyID:      1,
			Framework:  "openclaw",
			ConfigPath: "dup.md",
		})
		if err == nil {
			t.Error("expected error for duplicate key_id+framework")
		}
	})
}

func TestFrameworkMappingRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFrameworkMappingGormRepo(db)
	ctx := context.Background()

	t.Run("按 ID 获取", func(t *testing.T) {
		m, err := repo.Get(ctx, 1)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if m.Framework != "openclaw" {
			t.Errorf("expected 'openclaw', got %q", m.Framework)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := repo.Get(ctx, 999)
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestFrameworkMappingRepo_GetByKeyAndFramework(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFrameworkMappingGormRepo(db)
	ctx := context.Background()

	t.Run("按 key_id+framework 获取", func(t *testing.T) {
		m, err := repo.GetByKeyAndFramework(ctx, 1, "openclaw")
		if err != nil {
			t.Fatalf("GetByKeyAndFramework failed: %v", err)
		}
		if m.ConfigPath != "SOUL.md" {
			t.Errorf("expected 'SOUL.md', got %q", m.ConfigPath)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := repo.GetByKeyAndFramework(ctx, 999, "openclaw")
		if err == nil {
			t.Error("expected error for non-existent mapping")
		}
	})
}

func TestFrameworkMappingRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFrameworkMappingGormRepo(db)
	ctx := context.Background()

	t.Run("更新 config_path", func(t *testing.T) {
		m, _ := repo.Get(ctx, 1)
		m.ConfigPath = "NEW_SOUL.md"
		if err := repo.Update(ctx, m); err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		got, _ := repo.Get(ctx, 1)
		if got.ConfigPath != "NEW_SOUL.md" {
			t.Errorf("expected 'NEW_SOUL.md', got %q", got.ConfigPath)
		}
	})

	t.Run("更新不存在的记录失败", func(t *testing.T) {
		m := &model.FrameworkMapping{ID: 999, KeyID: 1, Framework: "ghost", ConfigPath: "x"}
		if err := repo.Update(ctx, m); err == nil {
			t.Error("expected error for non-existent record")
		}
	})
}

func TestFrameworkMappingRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFrameworkMappingGormRepo(db)
	ctx := context.Background()

	t.Run("删除已存在的记录", func(t *testing.T) {
		if err := repo.Delete(ctx, 1); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := repo.Get(ctx, 1); err == nil {
			t.Error("expected error after delete")
		}
	})
}

func TestFrameworkMappingRepo_ListByFramework(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFrameworkMappingGormRepo(db)
	ctx := context.Background()

	t.Run("openclaw 预置 6 条", func(t *testing.T) {
		mappings, total, err := repo.ListByFramework(ctx, "openclaw", 1, 10)
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

	t.Run("claude-code 预置 2 条", func(t *testing.T) {
		mappings, total, err := repo.ListByFramework(ctx, "claude-code", 1, 10)
		if err != nil {
			t.Fatalf("ListByFramework failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
		if len(mappings) != 2 {
			t.Errorf("expected 2 items, got %d", len(mappings))
		}
	})

	t.Run("不存在的框架返回空", func(t *testing.T) {
		mappings, total, _ := repo.ListByFramework(ctx, "nonexistent", 1, 10)
		if total != 0 {
			t.Errorf("expected 0, got %d", total)
		}
		if len(mappings) != 0 {
			t.Errorf("expected 0 items, got %d", len(mappings))
		}
	})
}

func TestFrameworkMappingRepo_ListByKeyID(t *testing.T) {
	db := setupTestDB(t)
	repo := NewFrameworkMappingGormRepo(db)
	ctx := context.Background()

	t.Run("system_prompt (key_id=1) 映射到 2 个框架", func(t *testing.T) {
		mappings, err := repo.ListByKeyID(ctx, 1)
		if err != nil {
			t.Fatalf("ListByKeyID failed: %v", err)
		}
		if len(mappings) != 2 {
			t.Errorf("expected 2, got %d", len(mappings))
		}
	})
}
