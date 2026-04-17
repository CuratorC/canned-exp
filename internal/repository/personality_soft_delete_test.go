package repository

import (
	"context"
	"testing"

	"canned-exp/internal/model"
)

func TestPersonalityGormRepo_SoftDelete(t *testing.T) {
	pRepo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "sd-agent"})

	t.Run("Delete 后 Get 返回 not found", func(t *testing.T) {
		id, _ := pRepo.Save(ctx, &model.Personality{
			AgentID: agentID, KeyID: 1, Value: "test",
		})
		if err := pRepo.Delete(ctx, id); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		_, err := pRepo.Get(ctx, id)
		if err == nil {
			t.Error("expected not found after soft delete")
		}
	})

	t.Run("Delete 后 ListByAgent 不包含已删除记录", func(t *testing.T) {
		id, _ := pRepo.Save(ctx, &model.Personality{
			AgentID: agentID, KeyID: 2, Value: "list-test",
		})
		pRepo.Delete(ctx, id)

		list, _, _ := pRepo.ListByAgent(ctx, agentID, 1, 100)
		for _, p := range list {
			if p.ID == id {
				t.Error("soft-deleted personality should not appear in ListByAgent")
			}
		}
	})

	t.Run("Delete 后记录仍存在（Unscoped 可查到）", func(t *testing.T) {
		id, _ := pRepo.Save(ctx, &model.Personality{
			AgentID: agentID, KeyID: 3, Value: "unscoped-test",
		})
		pRepo.Delete(ctx, id)

		var deletedAt string
		pRepo.db.Raw("SELECT deleted_at FROM personalities WHERE id = ?", id).Scan(&deletedAt)
		if deletedAt == "" {
			t.Error("expected deleted_at to be set, got empty")
		}
	})

	t.Run("Delete 后 Set 同 agent_id+key_id 成功", func(t *testing.T) {
		p := &model.Personality{
			AgentID: agentID, KeyID: 4, Value: "before-delete",
		}
		id, _ := pRepo.Upsert(ctx, p)
		pRepo.Delete(ctx, id)

		newID, err := pRepo.Upsert(ctx, &model.Personality{
			AgentID: agentID, KeyID: 4, Value: "after-delete",
		})
		if err != nil {
			t.Fatalf("expected Upsert after soft delete to succeed, got: %v", err)
		}
		if newID == 0 {
			t.Error("expected non-zero ID for new personality")
		}

		got, _ := pRepo.Get(ctx, newID)
		if got.Value != "after-delete" {
			t.Errorf("expected 'after-delete', got %q", got.Value)
		}
	})

	t.Run("Delete 后再次 Delete 不报错", func(t *testing.T) {
		id, _ := pRepo.Save(ctx, &model.Personality{
			AgentID: agentID, KeyID: 5, Value: "double-delete",
		})
		if err := pRepo.Delete(ctx, id); err != nil {
			t.Fatalf("first Delete failed: %v", err)
		}
		if err := pRepo.Delete(ctx, id); err != nil {
			t.Fatalf("second Delete failed: %v", err)
		}
	})
}
