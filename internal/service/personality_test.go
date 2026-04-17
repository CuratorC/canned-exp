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

func setupPersonalityService(t *testing.T) (*PersonalityService, *repository.AgentGormRepo) {
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

	pRepo := repository.NewPersonalityGormRepo(gormDB)
	pkRepo := repository.NewPersonalityKeyGormRepo(gormDB)
	agentRepo := repository.NewAgentGormRepo(gormDB)
	return NewPersonalityService(pRepo, pkRepo), agentRepo
}

func TestPersonalityService_Set(t *testing.T) {
	svc, agentRepo := setupPersonalityService(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	t.Run("首次 set 创建", func(t *testing.T) {
		id, err := svc.Set(ctx, &model.Personality{
			AgentID: agentID,
			KeyID:   1,
			Value:   "你好",
		})
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复 set 更新", func(t *testing.T) {
		id, err := svc.Set(ctx, &model.Personality{
			AgentID: agentID,
			KeyID:   1,
			Value:   "更新后的值",
		})
		if err != nil {
			t.Fatalf("Set upsert failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}

		got, _ := svc.Get(ctx, id)
		if got.Value != "更新后的值" {
			t.Errorf("expected '更新后的值', got %q", got.Value)
		}
	})

	t.Run("无效 key_id 被拒绝", func(t *testing.T) {
		_, err := svc.Set(ctx, &model.Personality{
			AgentID: agentID,
			KeyID:   999,
			Value:   "无效 key",
		})
		if err == nil {
			t.Error("expected error for non-existent key_id")
		}
	})

	t.Run("空 agent_id 被拒绝", func(t *testing.T) {
		_, err := svc.Set(ctx, &model.Personality{
			AgentID: 0,
			KeyID:   1,
			Value:   "无 agent",
		})
		if err == nil {
			t.Error("expected validation error for zero agent_id")
		}
	})

	t.Run("空 value 被拒绝", func(t *testing.T) {
		_, err := svc.Set(ctx, &model.Personality{
			AgentID: agentID,
			KeyID:   2,
			Value:   "",
		})
		if err == nil {
			t.Error("expected validation error for empty value")
		}
	})
}

func TestPersonalityService_Get(t *testing.T) {
	svc, agentRepo := setupPersonalityService(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	id, _ := svc.Set(ctx, &model.Personality{
		AgentID: agentID,
		KeyID:   1,
		Value:   "可获取",
	})

	t.Run("按 ID 获取", func(t *testing.T) {
		p, err := svc.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if p.Value != "可获取" {
			t.Errorf("expected '可获取', got %q", p.Value)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := svc.Get(ctx, 999)
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestPersonalityService_Update(t *testing.T) {
	svc, agentRepo := setupPersonalityService(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	id, _ := svc.Set(ctx, &model.Personality{
		AgentID: agentID,
		KeyID:   1,
		Value:   "原值",
	})

	t.Run("更新 value", func(t *testing.T) {
		p, _ := svc.Get(ctx, id)
		p.Value = "新值"
		if err := svc.Update(ctx, p); err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		got, _ := svc.Get(ctx, id)
		if got.Value != "新值" {
			t.Errorf("expected '新值', got %q", got.Value)
		}
	})

	t.Run("空 value 更新被拒绝", func(t *testing.T) {
		p, _ := svc.Get(ctx, id)
		p.Value = "   "
		if err := svc.Update(ctx, p); err == nil {
			t.Error("expected validation error for whitespace value")
		}
	})
}

func TestPersonalityService_Delete(t *testing.T) {
	svc, agentRepo := setupPersonalityService(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	id, _ := svc.Set(ctx, &model.Personality{
		AgentID: agentID,
		KeyID:   1,
		Value:   "待删除",
	})

	t.Run("删除", func(t *testing.T) {
		if err := svc.Delete(ctx, id); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := svc.Get(ctx, id); err == nil {
			t.Error("expected error after delete")
		}
	})
}

func TestPersonalityService_ListByAgent(t *testing.T) {
	svc, agentRepo := setupPersonalityService(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	agent2ID, _ := agentRepo.Save(ctx, &model.Agent{Name: "other"})

	svc.Set(ctx, &model.Personality{AgentID: agentID, KeyID: 1, Value: "v1"})
	svc.Set(ctx, &model.Personality{AgentID: agentID, KeyID: 2, Value: "v2"})
	svc.Set(ctx, &model.Personality{AgentID: agent2ID, KeyID: 1, Value: "other"})

	t.Run("只返回指定 agent", func(t *testing.T) {
		list, total, err := svc.ListByAgent(ctx, agentID, 1, 10)
		if err != nil {
			t.Fatalf("ListByAgent failed: %v", err)
		}
		if total != 2 {
			t.Errorf("expected 2, got %d", total)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 items, got %d", len(list))
		}
	})

	t.Run("默认分页", func(t *testing.T) {
		list, _, err := svc.ListByAgent(ctx, agentID, 0, 0)
		if err != nil {
			t.Fatalf("ListByAgent with defaults failed: %v", err)
		}
		if len(list) != 2 {
			t.Errorf("expected 2 items, got %d", len(list))
		}
	})
}
