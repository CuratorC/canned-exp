package agent

import (
	"context"
	"testing"

	"github.com/dromara/carbon/v2"
)

func closeDB(gormDB *GormRepo) {
	// no-op: 内存数据库随测试结束自动清理
}

func TestAgentRepo_SaveAndGet(t *testing.T) {
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	t.Run("保存并按ID获取", func(t *testing.T) {
		agent := &Agent{
			Name:        "Claude",
			Description: "AI coding assistant",
		}
		id, err := repo.Save(ctx, agent)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == "" {
			t.Fatal("expected non-empty ID from Save")
		}

		got, err := repo.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if got.ID != id {
			t.Errorf("ID mismatch: got %q, want %q", got.ID, id)
		}
		if got.Name != "Claude" {
			t.Errorf("Name mismatch: got %q, want %q", got.Name, "Claude")
		}
		if got.Description != "AI coding assistant" {
			t.Errorf("Description mismatch: got %q, want %q", got.Description, "AI coding assistant")
		}
		if got.CreatedAt.IsZero() {
			t.Error("CreatedAt should not be zero")
		}
		if got.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should not be zero")
		}
	})

	t.Run("获取不存在的ID返回错误", func(t *testing.T) {
		_, err := repo.Get(ctx, "nonexistent-id")
		if err == nil {
			t.Fatal("expected error for nonexistent ID, got nil")
		}
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got: %v", err)
		}
	})
}

func TestAgentRepo_Update(t *testing.T) {
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	t.Run("更新名称和描述", func(t *testing.T) {
		agent := &Agent{
			Name:        "Original",
			Description: "original desc",
		}
		id, _ := repo.Save(ctx, agent)

		agent.ID = id
		agent.Name = "Updated"
		agent.Description = "updated desc"
		if err := repo.Update(ctx, agent); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := repo.Get(ctx, id)
		if got.Name != "Updated" {
			t.Errorf("Name not updated: got %q, want %q", got.Name, "Updated")
		}
		if got.Description != "updated desc" {
			t.Errorf("Description not updated: got %q, want %q", got.Description, "updated desc")
		}
	})

	t.Run("更新不存在的ID返回错误", func(t *testing.T) {
		agent := &Agent{
			ID:   "nonexistent-id",
			Name: "Ghost",
		}
		err := repo.Update(ctx, agent)
		if err == nil {
			t.Fatal("expected error for updating nonexistent ID, got nil")
		}
	})
}

func TestAgentRepo_Delete(t *testing.T) {
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	t.Run("删除后获取返回错误", func(t *testing.T) {
		agent := &Agent{Name: "ToDelete"}
		id, _ := repo.Save(ctx, agent)

		if err := repo.Delete(ctx, id); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err := repo.Get(ctx, id)
		if err == nil {
			t.Fatal("expected ErrNotFound after delete, got nil")
		}
	})

	t.Run("删除不存在的ID不报错", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent-id")
		if err != nil {
			t.Errorf("delete of nonexistent ID should not error, got: %v", err)
		}
	})
}

func TestAgentRepo_List(t *testing.T) {
	repo, _ := setupTestRepo(t)
	ctx := context.Background()

	t.Run("分页返回Agent列表", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			a := &Agent{Name: "Agent"}
			repo.Save(ctx, a)
		}

		agents, total, err := repo.List(ctx, 1, 3)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 5 {
			t.Errorf("total mismatch: got %d, want 5", total)
		}
		if len(agents) != 3 {
			t.Errorf("page size mismatch: got %d agents, want 3", len(agents))
		}
	})

	t.Run("第二页与第一页不重叠", func(t *testing.T) {
		page1, _, _ := repo.List(ctx, 1, 3)
		page2, _, _ := repo.List(ctx, 2, 3)

		page1IDs := make(map[string]bool)
		for _, a := range page1 {
			page1IDs[a.ID] = true
		}
		for _, a := range page2 {
			if page1IDs[a.ID] {
				t.Errorf("agent %q appears in both page 1 and page 2", a.ID)
			}
		}
	})

	t.Run("空库返回空列表和total为0", func(t *testing.T) {
		emptyRepo, _ := setupTestRepo(t)

		agents, total, err := emptyRepo.List(ctx, 1, 20)
		if err != nil {
			t.Fatalf("List on empty DB failed: %v", err)
		}
		if total != 0 {
			t.Errorf("expected total 0 for empty DB, got %d", total)
		}
		if len(agents) != 0 {
			t.Errorf("expected empty list for empty DB, got %d agents", len(agents))
		}
	})
}

// 确保 carbon 包被使用
var _ = carbon.Now
