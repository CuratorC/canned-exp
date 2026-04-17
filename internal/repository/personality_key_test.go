package repository

import (
	"context"
	"testing"

	"canned-exp/internal/model"
)

func TestPersonalityKeyGormRepo_Save(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPersonalityKeyGormRepo(db)
	ctx := context.Background()

	t.Run("正常保存并返回 ID", func(t *testing.T) {
		pk := &model.PersonalityKey{
			KeyName:     "custom_key",
			Description: "自定义 key",
		}
		id, err := repo.Save(ctx, pk)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复 key_name 失败", func(t *testing.T) {
		// 预置 system_prompt 已在迁移中插入
		pk := &model.PersonalityKey{
			KeyName:     "system_prompt",
			Description: "重复的 key",
		}
		_, err := repo.Save(ctx, pk)
		if err == nil {
			t.Error("expected error for duplicate key_name, got nil")
		}
	})
}

func TestPersonalityKeyGormRepo_Get(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPersonalityKeyGormRepo(db)
	ctx := context.Background()

	t.Run("按 ID 获取", func(t *testing.T) {
		pk, err := repo.Get(ctx, 1)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if pk.KeyName != "system_prompt" {
			t.Errorf("expected key_name 'system_prompt', got %q", pk.KeyName)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := repo.Get(ctx, 999)
		if err == nil {
			t.Error("expected error for non-existent ID, got nil")
		}
	})
}

func TestPersonalityKeyGormRepo_GetByName(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPersonalityKeyGormRepo(db)
	ctx := context.Background()

	t.Run("按 key_name 获取", func(t *testing.T) {
		pk, err := repo.GetByName(ctx, "name")
		if err != nil {
			t.Fatalf("GetByName failed: %v", err)
		}
		if pk.KeyName != "name" {
			t.Errorf("expected key_name 'name', got %q", pk.KeyName)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := repo.GetByName(ctx, "nonexistent_key")
		if err == nil {
			t.Error("expected error for non-existent key_name, got nil")
		}
	})
}

func TestPersonalityKeyGormRepo_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPersonalityKeyGormRepo(db)
	ctx := context.Background()

	t.Run("更新描述", func(t *testing.T) {
		pk, _ := repo.Get(ctx, 1)
		pk.Description = "更新后的描述"
		if err := repo.Update(ctx, pk); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := repo.Get(ctx, 1)
		if got.Description != "更新后的描述" {
			t.Errorf("expected updated description, got %q", got.Description)
		}
	})

	t.Run("更新不存在的记录失败", func(t *testing.T) {
		pk := &model.PersonalityKey{ID: 999, KeyName: "ghost"}
		if err := repo.Update(ctx, pk); err == nil {
			t.Error("expected error for updating non-existent key, got nil")
		}
	})
}

func TestPersonalityKeyGormRepo_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPersonalityKeyGormRepo(db)
	ctx := context.Background()

	t.Run("删除已存在的记录", func(t *testing.T) {
		if err := repo.Delete(ctx, 1); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := repo.Get(ctx, 1); err == nil {
			t.Error("expected error after delete, got nil")
		}
	})
}

func TestPersonalityKeyGormRepo_List(t *testing.T) {
	db := setupTestDB(t)
	repo := NewPersonalityKeyGormRepo(db)
	ctx := context.Background()

	t.Run("列出所有预置 key", func(t *testing.T) {
		keys, total, err := repo.List(ctx, 1, 10)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 6 {
			t.Errorf("expected 6 preset keys, got %d", total)
		}
		if len(keys) != 6 {
			t.Errorf("expected 6 keys in page, got %d", len(keys))
		}
	})

	t.Run("分页", func(t *testing.T) {
		keys, _, err := repo.List(ctx, 1, 3)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if len(keys) != 3 {
			t.Errorf("expected 3 keys in page, got %d", len(keys))
		}
	})
}
