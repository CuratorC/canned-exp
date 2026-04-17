package repository

import (
	"context"
	"testing"

	"canned-exp/internal/model"
)

func TestAgentGormRepo_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentGormRepo(db)
	ctx := context.Background()

	t.Run("Delete 后 Get 返回 not found", func(t *testing.T) {
		id, _ := repo.Save(ctx, &model.Agent{Name: "soft-delete-test"})
		if err := repo.Delete(ctx, id); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		_, err := repo.Get(ctx, id)
		if err == nil {
			t.Error("expected not found after soft delete, got nil error")
		}
	})

	t.Run("Delete 后 List 不包含已删除记录", func(t *testing.T) {
		id, _ := repo.Save(ctx, &model.Agent{Name: "list-filter-test"})
		repo.Delete(ctx, id)

		agents, total, err := repo.List(ctx, 1, 100)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		for _, a := range agents {
			if a.ID == id {
				t.Error("soft-deleted agent should not appear in List")
			}
		}
		_ = total
	})

	t.Run("Delete 后记录仍存在（Unscoped 可查到）", func(t *testing.T) {
		id, _ := repo.Save(ctx, &model.Agent{Name: "unscoped-test"})
		repo.Delete(ctx, id)

		var deletedAt string
		db.Raw("SELECT deleted_at FROM agents WHERE id = ?", id).Scan(&deletedAt)
		if deletedAt == "" {
			t.Error("expected deleted_at to be set, got empty")
		}
	})

	t.Run("Delete 后可创建同名 Agent", func(t *testing.T) {
		id1, _ := repo.Save(ctx, &model.Agent{Name: "same-name"})
		repo.Delete(ctx, id1)

		id2, err := repo.Save(ctx, &model.Agent{Name: "same-name"})
		if err != nil {
			t.Fatalf("expected to create agent with same name after soft delete, got: %v", err)
		}
		if id2 == 0 {
			t.Error("expected non-zero ID for new agent")
		}
	})
}
