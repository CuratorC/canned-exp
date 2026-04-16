package agent

import (
	"context"
	"testing"
)

func testServiceSetup(t *testing.T) *Service {
	t.Helper()
	repo, gormDB := setupTestRepo(t)
	t.Cleanup(func() {
		sqlDB, _ := gormDB.DB()
		sqlDB.Close()
	})
	return NewService(repo)
}

func TestAgentService_Save(t *testing.T) {
	ctx := context.Background()

	t.Run("合法Agent保存成功", func(t *testing.T) {
		svc := testServiceSetup(t)
		a := &Agent{Name: "Claude", Description: "coding assistant"}
		id, err := svc.Save(ctx, a)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == "" {
			t.Fatal("expected non-empty ID")
		}
	})

	t.Run("无效Agent在Service层被拦截", func(t *testing.T) {
		svc := testServiceSetup(t)
		a := &Agent{Name: "", Description: ""}
		_, err := svc.Save(ctx, a)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})
}

func TestAgentService_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("获取已存在的Agent", func(t *testing.T) {
		svc := testServiceSetup(t)
		a := &Agent{Name: "Cursor", Description: "IDE assistant"}
		id, _ := svc.Save(ctx, a)

		got, err := svc.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if got.Name != "Cursor" {
			t.Errorf("Name mismatch: got %q, want %q", got.Name, "Cursor")
		}
	})

	t.Run("获取不存在返回错误", func(t *testing.T) {
		svc := testServiceSetup(t)
		_, err := svc.Get(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestAgentService_Update(t *testing.T) {
	ctx := context.Background()

	t.Run("更新成功", func(t *testing.T) {
		svc := testServiceSetup(t)
		a := &Agent{Name: "Old", Description: "old desc"}
		id, _ := svc.Save(ctx, a)

		a.ID = id
		a.Name = "New"
		a.Description = "new desc"
		if err := svc.Update(ctx, a); err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		got, _ := svc.Get(ctx, id)
		if got.Name != "New" {
			t.Errorf("Name not updated: got %q", got.Name)
		}
	})

	t.Run("更新不存在返回错误", func(t *testing.T) {
		svc := testServiceSetup(t)
		a := &Agent{ID: "ghost", Name: "Ghost"}
		err := svc.Update(ctx, a)
		if err == nil {
			t.Fatal("expected error for nonexistent ID, got nil")
		}
	})

	t.Run("更新无效数据被拦截", func(t *testing.T) {
		svc := testServiceSetup(t)
		a := &Agent{Name: "Valid"}
		id, _ := svc.Save(ctx, a)

		a.ID = id
		a.Name = ""
		err := svc.Update(ctx, a)
		if err == nil {
			t.Fatal("expected validation error, got nil")
		}
	})
}

func TestAgentService_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("删除成功", func(t *testing.T) {
		svc := testServiceSetup(t)
		a := &Agent{Name: "ToDelete"}
		id, _ := svc.Save(ctx, a)

		if err := svc.Delete(ctx, id); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		_, err := svc.Get(ctx, id)
		if err == nil {
			t.Fatal("expected ErrNotFound after delete")
		}
	})
}

func TestAgentService_List(t *testing.T) {
	ctx := context.Background()

	t.Run("分页返回结果", func(t *testing.T) {
		svc := testServiceSetup(t)
		for i := 0; i < 5; i++ {
			svc.Save(ctx, &Agent{Name: "Agent"})
		}

		agents, total, err := svc.List(ctx, 1, 3)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 5 {
			t.Errorf("total mismatch: got %d, want 5", total)
		}
		if len(agents) != 3 {
			t.Errorf("page size mismatch: got %d, want 3", len(agents))
		}
	})

	t.Run("page为0时默认为1", func(t *testing.T) {
		svc := testServiceSetup(t)
		svc.Save(ctx, &Agent{Name: "Agent"})

		agents, total, err := svc.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 1 {
			t.Errorf("total mismatch: got %d, want 1", total)
		}
		if len(agents) != 1 {
			t.Errorf("should return 1 agent, got %d", len(agents))
		}
	})

	t.Run("pageSize为0时使用默认值", func(t *testing.T) {
		svc := testServiceSetup(t)
		for i := 0; i < 25; i++ {
			svc.Save(ctx, &Agent{Name: "Agent"})
		}

		_, total, _ := svc.List(ctx, 1, 0)
		if total != 25 {
			t.Errorf("total mismatch: got %d, want 25", total)
		}
	})
}
