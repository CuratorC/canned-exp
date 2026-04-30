package controller

import (
	"context"
	"math"
	"strings"
	"testing"

	"canned-exp/internal/embedding"
	"canned-exp/internal/repository"
	"canned-exp/internal/service"
	"canned-exp/internal/renderer"
	"canned-exp/internal/vectorstore"

	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/migration"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mockEmbedding 用于 MCP 测试的 mock embedding
type mockEmbedding struct {
	dimensions int
}

func (m *mockEmbedding) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, text := range texts {
		results[i] = textToVec(text, m.dimensions)
	}
	return results, nil
}

func (m *mockEmbedding) Dimensions() int {
	return m.dimensions
}

func textToVec(text string, dims int) []float64 {
	vec := make([]float64, dims)
	for _, r := range text {
		idx := int(uint32(r)) % dims
		vec[idx] += 1.0
	}
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for j := range vec {
			vec[j] /= norm
		}
	}
	return vec
}

// testMCPServer 创建使用 mock Service 的 MCP Server 实例
func testMCPServer(t *testing.T) *Server {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm memory sqlite: %v", err)
	}

	// 通过 gocanned migration 初始化表结构
	dbDB := &database.DB{Gorm: gormDB, Driver: "sqlite"}
	runner := migration.NewRunner("main", dbDB)
	if err := runner.Run(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	vecStore, err := vectorstore.NewSQLiteVecStore(dbDB.SQL())
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}

	emb := &mockEmbedding{dimensions: 64}
	expRepo := repository.NewExperienceGormRepo(gormDB, vecStore, emb)
	expSvc := service.NewExperienceService(expRepo)

	agentRepo := repository.NewAgentGormRepo(gormDB)
	agentSvc := service.NewAgentService(agentRepo)

			pRepo := repository.NewPersonalityGormRepo(gormDB)
		pkRepo := repository.NewPersonalityKeyGormRepo(gormDB)
		pSvc := service.NewPersonalityService(pRepo, pkRepo)
		pkSvc := service.NewPersonalityKeyService(pkRepo)
		mRepo := repository.NewMemoryGormRepo(gormDB)
		mSvc := service.NewMemoryService(mRepo)

		return NewServer(expSvc, agentSvc, pSvc, pkSvc, mSvc, renderer.NewRegistry())
}

// --- 工具注册测试 ---

func TestMCPServer_ToolRegistration(t *testing.T) {
	server := testMCPServer(t)

	t.Run("注册了所有必要的工具", func(t *testing.T) {
		tools := server.ListTools()
		names := make(map[string]bool)
		for _, tool := range tools {
			names[tool.Name] = true
		}

		required := []string{
			"save_experience",
			"search_experiences",
			"get_experience",
			"update_experience",
			"delete_experience",
			"list_experiences",
		}
		for _, name := range required {
			if !names[name] {
				t.Errorf("缺少工具: %s", name)
			}
		}
	})

	t.Run("每个工具都有非空的描述", func(t *testing.T) {
		tools := server.ListTools()
		for _, tool := range tools {
			if tool.Description == "" {
				t.Errorf("工具 %q 缺少描述", tool.Name)
			}
		}
	})

	t.Run("save_experience 必填参数包含 content 和 tags", func(t *testing.T) {
		tools := server.ListTools()
		var saveTool *ToolDefinition
		for i := range tools {
			if tools[i].Name == "save_experience" {
				saveTool = &tools[i]
				break
			}
		}
		if saveTool == nil {
			t.Fatal("未找到 save_experience 工具")
		}
		required := saveTool.RequiredParams()
		hasContent := false
		hasTags := false
		for _, p := range required {
			if p == "content" {
				hasContent = true
			}
			if p == "tags" {
				hasTags = true
			}
		}
		if !hasContent {
			t.Error("save_experience 的 content 应为必填参数")
		}
		if !hasTags {
			t.Error("save_experience 的 tags 应为必填参数")
		}
	})

	t.Run("search_experiences 必填参数包含 query", func(t *testing.T) {
		tools := server.ListTools()
		var searchTool *ToolDefinition
		for i := range tools {
			if tools[i].Name == "search_experiences" {
				searchTool = &tools[i]
				break
			}
		}
		if searchTool == nil {
			t.Fatal("未找到 search_experiences 工具")
		}
		required := searchTool.RequiredParams()
		found := false
		for _, p := range required {
			if p == "query" {
				found = true
				break
			}
		}
		if !found {
			t.Error("search_experiences 的 query 应为必填参数")
		}
	})

	t.Run("get_experience 必填参数包含 id", func(t *testing.T) {
		tools := server.ListTools()
		var getTool *ToolDefinition
		for i := range tools {
			if tools[i].Name == "get_experience" {
				getTool = &tools[i]
				break
			}
		}
		if getTool == nil {
			t.Fatal("未找到 get_experience 工具")
		}
		required := getTool.RequiredParams()
		found := false
		for _, p := range required {
			if p == "id" {
				found = true
				break
			}
		}
		if !found {
			t.Error("get_experience 的 id 应为必填参数")
		}
	})

	t.Run("delete_experience 必填参数包含 id", func(t *testing.T) {
		tools := server.ListTools()
		var delTool *ToolDefinition
		for i := range tools {
			if tools[i].Name == "delete_experience" {
				delTool = &tools[i]
				break
			}
		}
		if delTool == nil {
			t.Fatal("未找到 delete_experience 工具")
		}
		required := delTool.RequiredParams()
		found := false
		for _, p := range required {
			if p == "id" {
				found = true
				break
			}
		}
		if !found {
			t.Error("delete_experience 的 id 应为必填参数")
		}
	})
}

// --- 工具调用测试 ---

func TestMCPServer_ToolCalls(t *testing.T) {
	server := testMCPServer(t)

	t.Run("save_experience 正常调用", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "Go 项目的测试文件应放在和源文件相同的目录",
			"title":   "Go 测试文件组织",
			"tags":    []interface{}{"go", "testing"},
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
		if result.Content == "" {
			t.Error("返回内容不应为空")
		}
	})

	t.Run("save_experience 缺少必填参数 content 时返回错误", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"title": "只有标题",
			"tags":  []interface{}{"test"},
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("缺少必填参数时应返回错误")
		}
	})

	t.Run("save_experience 缺少必填参数 tags 时返回错误", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "没有标签的经验",
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("缺少 tags 参数时应返回错误")
		}
	})

	t.Run("save_experience tags 为空数组时返回错误", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "空标签的经验",
			"tags":    []interface{}{},
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("tags 为空数组时应返回错误")
		}
	})

	t.Run("search_experiences 正常调用", func(t *testing.T) {
		server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "使用 SQLite 做本地存储时，注意单连接模式避免并发锁",
			"title":   "SQLite 使用经验",
			"tags":    []interface{}{"sqlite", "storage"},
			"agent_id": float64(0),
		})

		result, err := server.CallTool(context.Background(), "search_experiences", map[string]interface{}{
			"query": "SQLite 怎么用",
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})

	t.Run("search_experiences 缺少 query 时返回错误", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "search_experiences", map[string]interface{}{
			"top_k": 5,
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("缺少 query 参数时应返回错误")
		}
	})

	t.Run("get_experience 正常调用", func(t *testing.T) {
		saveResult, _ := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content":  "用于测试 Get 的经验",
			"tags":     []interface{}{"test"},
			"agent_id": float64(0),
		})
		id := saveResult.Content

		result, err := server.CallTool(context.Background(), "get_experience", map[string]interface{}{
			"id": id,
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})

	t.Run("get_experience 获取不存在的 ID 返回错误", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "get_experience", map[string]interface{}{
			"id": "nonexistent-id",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("获取不存在的 ID 应返回错误")
		}
	})

	t.Run("delete_experience 正常调用", func(t *testing.T) {
		saveResult, _ := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content":  "即将被删除的经验",
			"tags":     []interface{}{"test"},
			"agent_id": float64(0),
		})
		id := saveResult.Content

		result, err := server.CallTool(context.Background(), "delete_experience", map[string]interface{}{
			"id": id,
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})

	t.Run("list_experiences 正常调用", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "list_experiences", map[string]interface{}{
			"page":     1,
			"pageSize": 10,
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})

		t.Run("list_experiences 缺少 agent_id 时返回错误", func(t *testing.T) {
			result, err := server.CallTool(context.Background(), "list_experiences", map[string]interface{}{})
			if err != nil {
				t.Fatalf("CallTool() error: %v", err)
			}
			if !result.IsError {
				t.Error("缺少 agent_id 应返回错误")
			}
		})

	t.Run("save_experience 发现相似内容返回警告", func(t *testing.T) {
		// 先保存一条
		server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "使用 errors.Is 和 errors.As 做错误比较",
			"tags":    []interface{}{"go", "error"},
			"agent_id": float64(0),
		})

		// 保存完全相同的内容
		result, err := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "使用 errors.Is 和 errors.As 做错误比较",
			"tags":    []interface{}{"go", "error"},
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("去重不应返回 isError: %s", result.Content)
		}
		// 去重响应应包含警告信息，不应是纯 UUID
		if len(result.Content) < 10 {
			t.Errorf("去重响应应包含详细信息，实际: %s", result.Content)
		}
	})

	t.Run("save_experience force=true 强制保存相似内容", func(t *testing.T) {
		// 先保存一条
		server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "使用 sync.Pool 复用对象减少 GC 压力",
			"tags":    []interface{}{"go", "performance"},
			"agent_id": float64(0),
		})

		// 保存相同内容，带 force=true
		result, err := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content": "使用 sync.Pool 复用对象减少 GC 压力",
			"tags":    []interface{}{"go", "performance"},
			"force":   true,
			"agent_id": float64(0),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("force=true 应成功保存: %s", result.Content)
		}
	})

	t.Run("调用不存在的工具返回错误", func(t *testing.T) {
		result, err := server.CallTool(context.Background(), "nonexistent_tool", map[string]interface{}{})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("调用不存在的工具应返回错误")
		}
	})

	t.Run("update_experience 正常调用", func(t *testing.T) {
		saveResult, _ := server.CallTool(context.Background(), "save_experience", map[string]interface{}{
			"content":  "原始经验内容",
			"title":    "原始标题",
			"tags":     []interface{}{"test"},
			"agent_id": float64(0),
		})
		id := saveResult.Content

		result, err := server.CallTool(context.Background(), "update_experience", map[string]interface{}{
			"id":      id,
			"content": "更新后的经验内容",
			"title":   "更新后的标题",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})
}

// 确保 mockEmbedding 实现了接口
var _ embedding.Provider = (*mockEmbedding)(nil)

// --- Agent 工具 + agent_id 参数测试 ---

func TestMCPServer_AgentTools(t *testing.T) {
	server := testMCPServer(t)
	ctx := context.Background()

	t.Run("注册了 11 个工具", func(t *testing.T) {
		tools := server.ListTools()
		if len(tools) != 24 {
			t.Errorf("工具数量 = %d, want 24", len(tools))
		}
	})

	t.Run("save_experience 带 agent_id 存储", func(t *testing.T) {
		result, err := server.CallTool(ctx, "save_experience", map[string]interface{}{
			"content":  "带 AgentID 的经验",
			"tags":     []interface{}{"test"},
			"agent_id": float64(1),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
		// 通过 get 验证 agent_id
		id := result.Content
		got, _ := server.CallTool(ctx, "get_experience", map[string]interface{}{"id": id})
		if !contains(got.Content, "agent_id") {
			t.Errorf("经验应包含 agent_id=1, got: %s", got.Content)
		}
	})

	t.Run("search_experiences 带 agent_id 过滤", func(t *testing.T) {
		s := testMCPServer(t)
		s.CallTool(ctx, "save_experience", map[string]interface{}{
			"content":  "AgentA 的经验",
			"tags":     []interface{}{"test"},
			"agent_id": float64(1),
		})
		s.CallTool(ctx, "save_experience", map[string]interface{}{
			"content":  "AgentB 的经验",
			"tags":     []interface{}{"test"},
			"agent_id": float64(2),
		})

		result, err := s.CallTool(ctx, "search_experiences", map[string]interface{}{
			"query":    "经验",
			"agent_id": float64(1),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
		if contains(result.Content, "AgentB") {
			t.Error("搜索 agent-A 不应返回 AgentB 的经验")
		}
	})

	t.Run("list_experiences 带 agent_id 过滤", func(t *testing.T) {
		s := testMCPServer(t)
		s.CallTool(ctx, "save_experience", map[string]interface{}{
			"content":  "AgentX 经验",
			"tags":     []interface{}{"test"},
			"agent_id": float64(99),
		})
		s.CallTool(ctx, "save_experience", map[string]interface{}{
			"content": "全局经验",
			"tags":    []interface{}{"test"},
			"agent_id": float64(0),
		})

		result, err := s.CallTool(ctx, "list_experiences", map[string]interface{}{
			"agent_id": float64(99),
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})

	t.Run("register_agent 创建成功", func(t *testing.T) {
		result, err := server.CallTool(ctx, "register_agent", map[string]interface{}{
			"name":        "Claude",
			"description": "AI coding assistant",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
		if result.Content == "" {
			t.Error("应返回 Agent ID")
		}
	})

	t.Run("register_agent 缺少 name 报错", func(t *testing.T) {
		result, err := server.CallTool(ctx, "register_agent", map[string]interface{}{
			"description": "no name",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("缺少 name 应返回错误")
		}
	})

	t.Run("list_agents 返回已注册的 Agent", func(t *testing.T) {
		s := testMCPServer(t)
		s.CallTool(ctx, "register_agent", map[string]interface{}{
			"name": "TestAgent",
		})

		result, err := s.CallTool(ctx, "list_agents", map[string]interface{}{})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
		if !contains(result.Content, "TestAgent") {
			t.Errorf("应包含 TestAgent, got: %s", result.Content)
		}
	})

	t.Run("get_agent 返回正确信息", func(t *testing.T) {
		s := testMCPServer(t)
		regResult, _ := s.CallTool(ctx, "register_agent", map[string]interface{}{
			"name":        "GetTestAgent",
			"description": "for get test",
		})
		id := regResult.Content

		result, err := s.CallTool(ctx, "get_agent", map[string]interface{}{"id": id})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
		if !contains(result.Content, "GetTestAgent") {
			t.Errorf("应包含 Agent 名称, got: %s", result.Content)
		}
	})

	t.Run("update_agent 修改成功", func(t *testing.T) {
		s := testMCPServer(t)
		regResult, _ := s.CallTool(ctx, "register_agent", map[string]interface{}{
			"name": "OldName",
		})
		id := regResult.Content

		result, err := s.CallTool(ctx, "update_agent", map[string]interface{}{
			"id":          id,
			"name":        "NewName",
			"description": "updated",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}

		got, _ := s.CallTool(ctx, "get_agent", map[string]interface{}{"id": id})
		if !contains(got.Content, "NewName") {
			t.Errorf("更新后名称应为 NewName, got: %s", got.Content)
		}
	})

	t.Run("delete_agent 删除成功", func(t *testing.T) {
		s := testMCPServer(t)
		regResult, _ := s.CallTool(ctx, "register_agent", map[string]interface{}{
			"name": "ToDelete",
		})
		id := regResult.Content

		result, err := s.CallTool(ctx, "delete_agent", map[string]interface{}{"id": id})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}

		got, _ := s.CallTool(ctx, "get_agent", map[string]interface{}{"id": id})
		if !got.IsError {
			t.Error("删除后 get_agent 应返回错误")
		}
	})
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && len(sub) > 0 && strings.Contains(s, sub)))
}
