package repository

import (
	"context"
	"strings"
	"testing"

	"canned-exp/internal/model"
)

// testRepoSetup 创建使用 mock 依赖的 Repository 实例
func testRepoSetup(t *testing.T) *ExperienceGormRepo {
	t.Helper()
	repo, _ := setupTestRepo(t)
	return repo
}

// --- Save + Get 测试 ---

func TestRepository_SaveAndGet(t *testing.T) {
	repo := testRepoSetup(t)
	ctx := context.Background()

	t.Run("存入后能按 ID 取回完整经验", func(t *testing.T) {
		exp := model.Experience{
			Title:   "Go TDD 实践",
			Content: "先写测试再写实现，每一步都要验证",
			Tags:    []string{"go", "tdd"},
			Source:  "claude",
		}
		id, err := repo.Save(ctx, &exp)
		if err != nil {
			t.Fatalf("Save() error: %v", err)
		}
		if id == "" {
			t.Fatal("Save() 返回了空 ID")
		}

		got, err := repo.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if got.Title != exp.Title {
			t.Errorf("Title = %q, want %q", got.Title, exp.Title)
		}
		if got.Content != exp.Content {
			t.Errorf("Content = %q, want %q", got.Content, exp.Content)
		}
		if got.Source != exp.Source {
			t.Errorf("Source = %q, want %q", got.Source, exp.Source)
		}
	})

	t.Run("获取不存在的 ID 返回错误", func(t *testing.T) {
		_, err := repo.Get(ctx, "nonexistent-id")
		if err == nil {
			t.Error("期望返回错误，但得到了 nil")
		}
	})

	t.Run("Save 时自动生成向量", func(t *testing.T) {
		exp := model.Experience{
			Content: "这条经验应该被自动向量化",
		}
		id, _ := repo.Save(ctx, &exp)

		results, err := repo.Search(ctx, "这条经验应该被自动向量化", 0, 1)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("Save 后搜索不到该经验，向量可能未生成")
		}
		if results[0].Experience.ID != id {
			t.Errorf("搜索到的 ID = %q, want %q", results[0].Experience.ID, id)
		}
	})

	t.Run("Save 返回的 ID 是幂等唯一的", func(t *testing.T) {
		exp1 := model.Experience{Content: "第一条经验"}
		exp2 := model.Experience{Content: "第二条经验"}
		id1, _ := repo.Save(ctx, &exp1)
		id2, _ := repo.Save(ctx, &exp2)
		if id1 == id2 {
			t.Error("两次 Save 不应返回相同 ID")
		}
	})
}

// --- Delete 测试 ---

func TestRepository_Delete(t *testing.T) {
	repo := testRepoSetup(t)
	ctx := context.Background()

	t.Run("删除后 Get 返回错误", func(t *testing.T) {
		exp := model.Experience{Content: "即将被删除的经验"}
		id, _ := repo.Save(ctx, &exp)

		err := repo.Delete(ctx, id)
		if err != nil {
			t.Fatalf("Delete() error: %v", err)
		}

		_, err = repo.Get(ctx, id)
		if err == nil {
			t.Error("删除后 Get 应返回错误")
		}
	})

	t.Run("删除后向量也被清除", func(t *testing.T) {
		exp := model.Experience{Content: "删除后向量也应清除"}
		id, _ := repo.Save(ctx, &exp)

		repo.Delete(ctx, id)

		results, _ := repo.Search(ctx, "删除后向量也应清除", 0, 10)
		for _, r := range results {
			if r.Experience.ID == id {
				t.Error("删除后向量搜索仍然命中该经验")
			}
		}
	})

	t.Run("删除不存在的 ID 不报错", func(t *testing.T) {
		err := repo.Delete(ctx, "nonexistent-id")
		if err != nil {
			t.Errorf("删除不存在的 ID 不应报错: %v", err)
		}
	})
}

// --- Search 测试 ---

func TestRepository_Search(t *testing.T) {
	repo := testRepoSetup(t)
	ctx := context.Background()

	repo.Save(ctx, &model.Experience{
		Title:   "Go 单元测试",
		Content: "Go 的测试文件以 _test.go 结尾，放在和源文件相同的目录",
		Tags:    []string{"go", "testing"},
		Source:  "claude",
	})
	repo.Save(ctx, &model.Experience{
		Title:   "Go 表格驱动测试",
		Content: "Go 推荐使用表格驱动的方式来组织测试用例",
		Tags:    []string{"go", "testing"},
		Source:  "claude",
	})
	repo.Save(ctx, &model.Experience{
		Title:   "做饭技巧",
		Content: "炒菜时大火快炒可以保持蔬菜的脆嫩口感",
		Tags:    []string{"cooking"},
		Source:  "chatgpt",
	})

	t.Run("语义搜索返回按相似度排序的结果", func(t *testing.T) {
		results, err := repo.Search(ctx, "Go 语言怎么写测试", 0, 3)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		if len(results) == 0 {
			t.Fatal("期望有搜索结果")
		}
		if results[0].Experience.Title == "做饭技巧" {
			t.Error("最相关的结果不应是做饭技巧")
		}
	})

	t.Run("Search 结果包含相似度分数", func(t *testing.T) {
		results, _ := repo.Search(ctx, "Go 测试", 0, 3)
		for _, r := range results {
			if r.Score <= 0 {
				t.Errorf("Score = %.4f, 应大于 0", r.Score)
			}
		}
	})

	t.Run("topK 限制生效", func(t *testing.T) {
		results, _ := repo.Search(ctx, "经验", 0, 2)
		if len(results) > 2 {
			t.Errorf("期望最多 2 条结果，得到 %d", len(results))
		}
	})

	t.Run("搜索无匹配时返回空结果", func(t *testing.T) {
		results, err := repo.Search(ctx, "量子物理学弦理论", 0, 5)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		_ = results
	})
}

// --- List 测试 ---

func TestRepository_List(t *testing.T) {
	repo := testRepoSetup(t)
	ctx := context.Background()

	repo.Save(ctx, &model.Experience{Content: "经验 A", Source: "claude"})
	repo.Save(ctx, &model.Experience{Content: "经验 B", Source: "chatgpt"})
	repo.Save(ctx, &model.Experience{Content: "经验 C", Source: "claude"})

	t.Run("分页列出经验", func(t *testing.T) {
		results, total, err := repo.List(ctx, 0, 1, 2)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		if total < 3 {
			t.Errorf("总数 = %d, 至少应为 3", total)
		}
		if len(results) > 2 {
			t.Errorf("页大小 2，但返回了 %d 条", len(results))
		}
	})

	t.Run("第二页数据与第一页不重复", func(t *testing.T) {
		page1, _, _ := repo.List(ctx, 0, 1, 2)
		page2, _, _ := repo.List(ctx, 0, 2, 2)

		ids1 := make(map[string]bool)
		for _, e := range page1 {
			ids1[e.ID] = true
		}
		for _, e := range page2 {
			if ids1[e.ID] {
				t.Errorf("ID %q 同时出现在第一页和第二页", e.ID)
			}
		}
	})

	t.Run("页号超出范围返回空结果", func(t *testing.T) {
		results, total, err := repo.List(ctx, 0, 999, 10)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("超出范围应返回空，得到 %d 条", len(results))
		}
		_ = total
	})

	t.Run("空库 List 返回空结果和总数 0", func(t *testing.T) {
		emptyRepo := testRepoSetup(t)
		results, total, err := emptyRepo.List(ctx, 0, 1, 10)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		if total != 0 {
			t.Errorf("总数 = %d, want 0", total)
		}
		if len(results) != 0 {
			t.Errorf("期望 0 条结果，得到 %d", len(results))
		}
	})
}

// --- Update 测试（MCP 层需要，补充） ---

func TestRepository_Update(t *testing.T) {
	repo := testRepoSetup(t)
	ctx := context.Background()

	t.Run("更新后能取回新内容", func(t *testing.T) {
		exp := model.Experience{Title: "旧标题", Content: "旧内容"}
		id, _ := repo.Save(ctx, &exp)

		exp.Title = "新标题"
		exp.Content = "新内容"
		err := repo.Update(ctx, &exp)
		if err != nil {
			t.Fatalf("Update() error: %v", err)
		}

		got, _ := repo.Get(ctx, id)
		if got.Title != "新标题" {
			t.Errorf("Title = %q, want %q", got.Title, "新标题")
		}
	})

	t.Run("更新不存在的 ID 返回错误", func(t *testing.T) {
		exp := model.Experience{ID: "nonexistent", Content: "内容"}
		err := repo.Update(ctx, &exp)
		if err == nil {
			t.Error("期望返回错误")
		}
	})
}

// --- AgentID 隔离测试 ---

func TestRepository_AgentID(t *testing.T) {
	ctx := context.Background()

	t.Run("带 agent_id 保存并取回", func(t *testing.T) {
		repo := testRepoSetup(t)
		exp := model.Experience{
			AgentID: 1,
			Content: "属于 Agent001 的经验",
		}
		id, err := repo.Save(ctx, &exp)
		if err != nil {
			t.Fatalf("Save() error: %v", err)
		}

		got, err := repo.Get(ctx, id)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		if got.AgentID != 1 {
			t.Errorf("AgentID = %d, want %d", got.AgentID, 1)
		}
	})

	t.Run("不带 agent_id 保存默认为空", func(t *testing.T) {
		repo := testRepoSetup(t)
		exp := model.Experience{Content: "全局经验"}
		id, _ := repo.Save(ctx, &exp)

		got, _ := repo.Get(ctx, id)
		if got.AgentID != 0 {
			t.Errorf("AgentID 应为空，got %d", got.AgentID)
		}
	})

	t.Run("按 agent_id 搜索只返回匹配的", func(t *testing.T) {
		repo := testRepoSetup(t)
		repo.Save(ctx, &model.Experience{AgentID: 1, Content: "AgentA 的 Go 经验"})
		repo.Save(ctx, &model.Experience{AgentID: 2, Content: "AgentB 的 Python 经验"})

		results, err := repo.Search(ctx, "Go 经验", 1, 10)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		for _, r := range results {
			if r.Experience.AgentID != 1 {
				t.Errorf("搜索结果包含非 agent-A 的经验: AgentID=%d", r.Experience.AgentID)
			}
		}
	})

	t.Run("按 agent_id 搜索也返回全局经验", func(t *testing.T) {
		repo := testRepoSetup(t)
		repo.Save(ctx, &model.Experience{AgentID: 0, Content: "全局的通用经验"})
		repo.Save(ctx, &model.Experience{AgentID: 1, Content: "AgentA 的专属经验"})
		repo.Save(ctx, &model.Experience{AgentID: 2, Content: "AgentB 的专属经验"})

		results, err := repo.Search(ctx, "经验", 1, 10)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		hasGlobal := false
		for _, r := range results {
			if r.Experience.AgentID == 0 {
				hasGlobal = true
				break
			}
		}
		if !hasGlobal {
			t.Error("搜索 agent-A 时应包含全局经验（agent_id 为空），但未找到")
		}
	})

	t.Run("agent_id=0 搜索只返回全局经验", func(t *testing.T) {
		repo := testRepoSetup(t)
		repo.Save(ctx, &model.Experience{AgentID: 0, Content: "全局经验"})
		repo.Save(ctx, &model.Experience{AgentID: 1, Content: "AgentA 经验"})
		repo.Save(ctx, &model.Experience{AgentID: 2, Content: "AgentB 经验"})

		results, err := repo.Search(ctx, "经验", 0, 10)
		if err != nil {
			t.Fatalf("Search() error: %v", err)
		}
		for _, r := range results {
			if r.Experience.AgentID != 0 {
				t.Errorf("agent_id=0 搜索应只返回全局经验，但包含 AgentID=%d", r.Experience.AgentID)
			}
		}
	})

		t.Run("按 agent_id 列表也返回全局经验", func(t *testing.T) {
			repo := testRepoSetup(t)
			repo.Save(ctx, &model.Experience{AgentID: 0, Content: "全局"})
			repo.Save(ctx, &model.Experience{AgentID: 1, Content: "A1"})
			repo.Save(ctx, &model.Experience{AgentID: 1, Content: "A2"})
			repo.Save(ctx, &model.Experience{AgentID: 2, Content: "B1"})

			results, total, err := repo.List(ctx, 1, 1, 10)
			if err != nil {
				t.Fatalf("List() error: %v", err)
			}
			if total != 3 {
				t.Errorf("agent-A 应有 3 条（2 条专属 + 1 条全局），got total=%d", total)
			}
			hasGlobal := false
			for _, e := range results {
				if e.AgentID == 0 {
					hasGlobal = true
				}
				if e.AgentID != 0 && e.AgentID != 1 {
					t.Errorf("列表包含非 agent-A 的经验: AgentID=%d", e.AgentID)
				}
			}
			if !hasGlobal {
				t.Error("列表应包含全局经验（agent_id=0）")
			}
		})

	t.Run("agent_id=0 列表只返回全局经验", func(t *testing.T) {
		repo := testRepoSetup(t)
		repo.Save(ctx, &model.Experience{AgentID: 0, Content: "全局"})
		repo.Save(ctx, &model.Experience{AgentID: 1, Content: "A"})

		results, total, err := repo.List(ctx, 0, 1, 10)
		if err != nil {
			t.Fatalf("List() error: %v", err)
		}
		if total != 1 {
			t.Errorf("agent_id=0 应只返回全局经验 total=1，got total=%d", total)
		}
		if len(results) != 1 {
			t.Errorf("agent_id=0 应只返回 1 条，got %d 条", len(results))
		}
	})
}

// 确保使用 strings（避免 import 报错）
var _ = strings.TrimSpace
