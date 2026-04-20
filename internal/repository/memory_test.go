package repository

import (
	"context"
	"testing"

	"canned-exp/internal/model"
)

func setupMemoryRepo(t *testing.T) (*MemoryGormRepo, *AgentGormRepo) {
	t.Helper()
	db := setupTestDB(t)
	return NewMemoryGormRepo(db), NewAgentGormRepo(db)
}

func TestMemoryGormRepo_Save(t *testing.T) {
	repo, agentRepo := setupMemoryRepo(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	t.Run("正常保存", func(t *testing.T) {
		id, err := repo.Save(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "rules/test",
			Content: "测试内容",
		})
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复 agent_id+path 失败", func(t *testing.T) {
		_, err := repo.Save(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "rules/test",
			Content: "重复",
		})
		if err == nil {
			t.Error("expected error for duplicate agent_id+path")
		}
	})
}

func TestMemoryGormRepo_GetByAgentAndPath(t *testing.T) {
	repo, agentRepo := setupMemoryRepo(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "daily/2026-04-18", Content: "日记"})

	t.Run("按路径获取", func(t *testing.T) {
		m, err := repo.GetByAgentAndPath(ctx, agentID, "daily/2026-04-18")
		if err != nil {
			t.Fatalf("GetByAgentAndPath failed: %v", err)
		}
		if m.Content != "日记" {
			t.Errorf("expected '日记', got %q", m.Content)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := repo.GetByAgentAndPath(ctx, agentID, "nonexistent")
		if err == nil {
			t.Error("expected error for non-existent path")
		}
	})
}

func TestMemoryGormRepo_Upsert(t *testing.T) {
	repo, agentRepo := setupMemoryRepo(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	t.Run("首次插入", func(t *testing.T) {
		id, err := repo.Upsert(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "rules/affection",
			Content: "初始内容",
		})
		if err != nil {
			t.Fatalf("Upsert insert failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复时更新", func(t *testing.T) {
		id, err := repo.Upsert(ctx, &model.Memory{
			AgentID: agentID,
			Path:    "rules/affection",
			Content: "更新内容",
		})
		if err != nil {
			t.Fatalf("Upsert update failed: %v", err)
		}
		got, _ := repo.GetByAgentAndPath(ctx, agentID, "rules/affection")
		if got.Content != "更新内容" {
			t.Errorf("expected '更新内容', got %q", got.Content)
		}
		if got.ID != id {
			t.Errorf("expected ID %d, got %d", id, got.ID)
		}
	})
}

func TestMemoryGormRepo_Delete(t *testing.T) {
	repo, agentRepo := setupMemoryRepo(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "temp", Content: "待删除"})

	t.Run("按路径删除", func(t *testing.T) {
		if err := repo.Delete(ctx, agentID, "temp"); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := repo.GetByAgentAndPath(ctx, agentID, "temp"); err == nil {
			t.Error("expected error after delete")
		}
	})
}

func TestMemoryGormRepo_ListByAgent(t *testing.T) {
	repo, agentRepo := setupMemoryRepo(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "rules/a", Content: "规则A"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "rules/b", Content: "规则B"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "daily/2026-04-18", Content: "日记"})

	t.Run("列出全部", func(t *testing.T) {
		list, total, err := repo.ListByAgent(ctx, agentID, "", 1, 10)
		if err != nil {
			t.Fatalf("ListByAgent failed: %v", err)
		}
		if total != 3 {
			t.Errorf("expected 3, got %d", total)
		}
		if len(list) != 3 {
			t.Errorf("expected 3 items, got %d", len(list))
		}
	})

	t.Run("路径前缀过滤", func(t *testing.T) {
		_, total, err := repo.ListByAgent(ctx, agentID, "rules/", 1, 10)
		if err != nil {
			t.Fatalf("ListByAgent with prefix failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
	})
}

func TestMemoryGormRepo_ListByDateRange(t *testing.T) {
	repo, agentRepo := setupMemoryRepo(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "d1", Content: "4月1日", MemoryDate: "2026-04-01"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "d2", Content: "4月15日", MemoryDate: "2026-04-15"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "d3", Content: "无日期"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "d4", Content: "4月20日", MemoryDate: "2026-04-20"})

	t.Run("日期范围过滤", func(t *testing.T) {
		list, total, err := repo.ListByDateRange(ctx, agentID, "2026-04-10", "2026-04-18", 1, 10)
		if err != nil {
			t.Fatalf("ListByDateRange failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1, got %d", total)
		}
		if len(list) != 1 {
			t.Errorf("expected 1 item, got %d", len(list))
		}
	})

	t.Run("只指定起始日期", func(t *testing.T) {
		_, total, err := repo.ListByDateRange(ctx, agentID, "2026-04-15", "", 1, 10)
		if err != nil {
			t.Fatalf("ListByDateRange failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
	})
}

func TestMemoryGormRepo_Search(t *testing.T) {
	repo, agentRepo := setupMemoryRepo(t)
	ctx := context.Background()
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "r1", Title: "健身规则", Content: "每周至少2次"})
	repo.Save(ctx, &model.Memory{AgentID: agentID, Path: "r2", Title: "作息规则", Content: "23点前睡觉"})

	t.Run("搜索内容", func(t *testing.T) {
		_, total, err := repo.Search(ctx, agentID, "健身", 1, 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if total != 1 {
			t.Errorf("expected 1, got %d", total)
		}
	})

	t.Run("搜索标题", func(t *testing.T) {
		list, total, err := repo.Search(ctx, agentID, "规则", 1, 10)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 items, got %d", len(list))
		}
	})
}
