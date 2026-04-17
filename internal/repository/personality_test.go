package repository

import (
	"context"
	"testing"

	"canned-exp/internal/model"
)

func setupPersonalityRepo(t *testing.T) (*PersonalityGormRepo, *PersonalityKeyGormRepo, *AgentGormRepo) {
	t.Helper()
	db := setupTestDB(t)
	return NewPersonalityGormRepo(db), NewPersonalityKeyGormRepo(db), NewAgentGormRepo(db)
}

func TestPersonalityGormRepo_Save(t *testing.T) {
	repo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	// 先创建一个 Agent
	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	t.Run("正常保存", func(t *testing.T) {
		p := &model.Personality{
			AgentID: agentID,
			KeyID:   1, // system_prompt 预置 key
			Value:   "你是一个有帮助的助手",
			Type:    "string",
		}
		id, err := repo.Save(ctx, p)
		if err != nil {
			t.Fatalf("Save failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复 agent_id+key_id 失败", func(t *testing.T) {
		p := &model.Personality{
			AgentID: agentID,
			KeyID:   1,
			Value:   "重复",
		}
		_, err := repo.Save(ctx, p)
		if err == nil {
			t.Error("expected error for duplicate agent_id+key_id")
		}
	})
}

func TestPersonalityGormRepo_Get(t *testing.T) {
	repo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	savedID, _ := repo.Save(ctx, &model.Personality{
		AgentID: agentID,
		KeyID:   1,
		Value:   "测试值",
	})

	t.Run("按 ID 获取", func(t *testing.T) {
		p, err := repo.Get(ctx, savedID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if p.Value != "测试值" {
			t.Errorf("expected '测试值', got %q", p.Value)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := repo.Get(ctx, 999)
		if err == nil {
			t.Error("expected error for non-existent ID")
		}
	})
}

func TestPersonalityGormRepo_GetByAgentAndKey(t *testing.T) {
	repo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	repo.Save(ctx, &model.Personality{
		AgentID: agentID,
		KeyID:   1,
		Value:   "通过 agent+key 查找",
	})

	t.Run("按 agent_id+key_id 获取", func(t *testing.T) {
		p, err := repo.GetByAgentAndKey(ctx, agentID, 1)
		if err != nil {
			t.Fatalf("GetByAgentAndKey failed: %v", err)
		}
		if p.Value != "通过 agent+key 查找" {
			t.Errorf("expected '通过 agent+key 查找', got %q", p.Value)
		}
	})

	t.Run("不存在返回错误", func(t *testing.T) {
		_, err := repo.GetByAgentAndKey(ctx, agentID, 999)
		if err == nil {
			t.Error("expected error for non-existent combination")
		}
	})
}

func TestPersonalityGormRepo_Upsert(t *testing.T) {
	repo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})

	t.Run("首次插入", func(t *testing.T) {
		p := &model.Personality{
			AgentID: agentID,
			KeyID:   2, // name key
			Value:   "初次值",
		}
		id, err := repo.Upsert(ctx, p)
		if err != nil {
			t.Fatalf("Upsert insert failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}
	})

	t.Run("重复时更新", func(t *testing.T) {
		p := &model.Personality{
			AgentID: agentID,
			KeyID:   2,
			Value:   "更新值",
		}
		id, err := repo.Upsert(ctx, p)
		if err != nil {
			t.Fatalf("Upsert update failed: %v", err)
		}
		if id == 0 {
			t.Error("expected non-zero ID")
		}

		got, _ := repo.GetByAgentAndKey(ctx, agentID, 2)
		if got.Value != "更新值" {
			t.Errorf("expected '更新值' after upsert, got %q", got.Value)
		}
	})
}

func TestPersonalityGormRepo_Update(t *testing.T) {
	repo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	savedID, _ := repo.Save(ctx, &model.Personality{
		AgentID: agentID,
		KeyID:   1,
		Value:   "原值",
	})

	t.Run("更新 value", func(t *testing.T) {
		p, _ := repo.Get(ctx, savedID)
		p.Value = "新值"
		if err := repo.Update(ctx, p); err != nil {
			t.Fatalf("Update failed: %v", err)
		}
		got, _ := repo.Get(ctx, savedID)
		if got.Value != "新值" {
			t.Errorf("expected '新值', got %q", got.Value)
		}
	})

	t.Run("更新不存在的记录失败", func(t *testing.T) {
		p := &model.Personality{ID: 999, AgentID: agentID, KeyID: 1, Value: "ghost"}
		if err := repo.Update(ctx, p); err == nil {
			t.Error("expected error for non-existent record")
		}
	})
}

func TestPersonalityGormRepo_Delete(t *testing.T) {
	repo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	savedID, _ := repo.Save(ctx, &model.Personality{
		AgentID: agentID,
		KeyID:   1,
		Value:   "待删除",
	})

	t.Run("删除已存在的记录", func(t *testing.T) {
		if err := repo.Delete(ctx, savedID); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		if _, err := repo.Get(ctx, savedID); err == nil {
			t.Error("expected error after delete")
		}
	})
}

func TestPersonalityGormRepo_ListByAgent(t *testing.T) {
	repo, _, agentRepo := setupPersonalityRepo(t)
	ctx := context.Background()

	agentID, _ := agentRepo.Save(ctx, &model.Agent{Name: "test-agent"})
	agent2ID, _ := agentRepo.Save(ctx, &model.Agent{Name: "other-agent"})

	repo.Save(ctx, &model.Personality{AgentID: agentID, KeyID: 1, Value: "值1"})
	repo.Save(ctx, &model.Personality{AgentID: agentID, KeyID: 2, Value: "值2"})
	repo.Save(ctx, &model.Personality{AgentID: agent2ID, KeyID: 1, Value: "其他agent"})

	t.Run("只返回指定 agent 的记录", func(t *testing.T) {
		list, total, err := repo.ListByAgent(ctx, agentID, 1, 10)
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

	t.Run("空 agent 无记录", func(t *testing.T) {
		list, total, _ := repo.ListByAgent(ctx, 999, 1, 10)
		if total != 0 {
			t.Errorf("expected 0, got %d", total)
		}
		if len(list) != 0 {
			t.Errorf("expected 0 items, got %d", len(list))
		}
	})
}
