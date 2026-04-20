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

func setupMemoryService(t *testing.T) (*MemoryService, *repository.AgentGormRepo) {
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

	mRepo := repository.NewMemoryGormRepo(gormDB)
	agentRepo := repository.NewAgentGormRepo(gormDB)
	return NewMemoryService(mRepo), agentRepo
}

func TestMemoryService_Set(t *testing.T) {
	svc, agentRepo := setupMemoryService(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	t.Run("首次 set 创建", func(t *testing.T) {
		id, err := svc.Set(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "rules/test",
			Content: "测试内容",
		})
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复 set 更新", func(t *testing.T) {
		id, err := svc.Set(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "rules/test",
			Content: "更新内容",
		})
		if err != nil {
			t.Fatalf("Set upsert failed: %v", err)
		}
		got, _ := svc.GetByPath(ctx, agentID, "rules/test")
		if got.Content != "更新内容" {
			t.Errorf("expected '更新内容', got %q", got.Content)
		}
		if got.ID != id {
			t.Errorf("expected ID %d, got %d", id, got.ID)
		}
	})

	t.Run("空 agent_id 被拒绝", func(t *testing.T) {
		_, err := svc.Set(ctx, &model.Memory{
			AgentID: 0,
			Path:    "rules/test",
			Content: "无 agent",
		})
		if err == nil {
			t.Error("expected validation error for zero agent_id")
		}
	})

	t.Run("空 path 被拒绝", func(t *testing.T) {
		_, err := svc.Set(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "   ",
			Content: "内容",
		})
		if err == nil {
			t.Error("expected validation error for empty path")
		}
	})

	t.Run("空 content 被拒绝", func(t *testing.T) {
		_, err := svc.Set(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "rules/empty",
			Content: "   ",
		})
		if err == nil {
			t.Error("expected validation error for empty content")
		}
	})

	t.Run("path 超长被拒绝", func(t *testing.T) {
		longPath := ""
		for i := 0; i < 201; i++ {
			longPath += "a"
		}
		_, err := svc.Set(ctx, &model.Memory{
			AgentID: agentID,
			Path:    longPath,
			Content: "内容",
		})
		if err == nil {
			t.Error("expected validation error for long path")
		}
	})
}

func TestMemoryService_Get(t *testing.T) {
	svc, agentRepo := setupMemoryService(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	id, _ := svc.Set(ctx, &model.Memory{
		AgentID: agentID,
		Path:    "rules/get",
		Content: "可获取",
	})

	t.Run("按 ID 获取", func(t *testing.T) {
		m, err := svc.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if m.Content != "可获取" {
			t.Errorf("expected '可获取', got %q", m.Content)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := svc.Get(ctx, 999)
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestMemoryService_GetByPath(t *testing.T) {
	svc, agentRepo := setupMemoryService(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	svc.Set(ctx, &model.Memory{
		AgentID: agentID,
		Path:    "daily/2026-04-18",
		Content: "日记",
	})

	t.Run("按路径获取", func(t *testing.T) {
		m, err := svc.GetByPath(ctx, agentID, "daily/2026-04-18")
		if err != nil {
			t.Fatalf("GetByPath failed: %v", err)
		}
		if m.Content != "日记" {
			t.Errorf("expected '日记', got %q", m.Content)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := svc.GetByPath(ctx, agentID, "nonexistent")
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})
}

func TestMemoryService_Delete(t *testing.T) {
	svc, agentRepo := setupMemoryService(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	svc.Set(ctx, &model.Memory{
		AgentID: agentID,
		Path:    "temp",
		Content: "待删除",
	})

	t.Run("按路径删除", func(t *testing.T) {
		if err := svc.Delete(ctx, agentID, "temp"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := svc.GetByPath(ctx, agentID, "temp"); err == nil {
			t.Error("expected error after delete")
		}
	})
}

func TestMemoryService_List(t *testing.T) {
	svc, agentRepo := setupMemoryService(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "rules/a", Content: "规则A"})
	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "rules/b", Content: "规则B"})
	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "daily/1", Content: "日记"})

	t.Run("列出全部", func(t *testing.T) {
		_, total, err := svc.List(ctx, agentID, "", 1, 10)
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected 3, got %d", total)
		}
	})

	t.Run("路径前缀过滤", func(t *testing.T) {
		_, total, err := svc.List(ctx, agentID, "rules/", 1, 10)
		if err != nil {
			t.Fatalf("List with prefix failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
	})

	t.Run("默认分页", func(t *testing.T) {
		list, _, err := svc.List(ctx, agentID, "", 0, 0)
		if err != nil {
			t.Fatalf("List with defaults failed: %v", err)
		}
		if len(list) != 3 {
			t.Errorf("expected 3 items, got %d", len(list))
		}
	})
}

func TestMemoryService_ListByDateRange(t *testing.T) {
	svc, agentRepo := setupMemoryService(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "d1", Content: "4月1日", MemoryDate: "2026-04-01"})
	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "d2", Content: "4月15日", MemoryDate: "2026-04-15"})
	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "d3", Content: "4月20日", MemoryDate: "2026-04-20"})

	t.Run("日期范围过滤", func(t *testing.T) {
		_, total, err := svc.ListByDateRange(ctx, agentID, "2026-04-10", "2026-04-18", 1, 10)
		if err != nil {
			t.Fatalf("ListByDateRange failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1, got %d", total)
		}
	})

	t.Run("只指定起始日期", func(t *testing.T) {
		_, total, err := svc.ListByDateRange(ctx, agentID, "2026-04-15", "", 1, 10)
		if err != nil {
			t.Fatalf("ListByDateRange failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
	})
}

func TestMemoryService_Search(t *testing.T) {
	svc, agentRepo := setupMemoryService(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "r1", Title: "健身规则", Content: "每周至少2次"})
	svc.Set(ctx, &model.Memory{AgentID: agentID, Path: "r2", Title: "作息规则", Content: "23点前睡觉"})

	t.Run("搜索内容", func(t *testing.T) {
		_, total, err := svc.Search(ctx, agentID, "健身", 1, 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1, got %d", total)
		}
	})

	t.Run("搜索标题", func(t *testing.T) {
		_, total, err := svc.Search(ctx, agentID, "规则", 1, 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
	})
}
