package mcp

import (
	"context"
	"database/sql"
	"math"
	"testing"

	"canned-exp/internal/experience"
	"canned-exp/internal/experience/embedding"
	"canned-exp/internal/experience/vectorstore"

	_ "modernc.org/sqlite"
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

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open memory sqlite: %v", err)
	}

	vecStore, err := vectorstore.NewSQLiteVecStore(db)
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}

	emb := &mockEmbedding{dimensions: 64}

	repo, err := experience.NewSQLiteRepo(db, vecStore, emb)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	svc := experience.NewService(repo)
	return NewServer(svc)
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
		result, err := server.CallTool("save_experience", map[string]interface{}{
			"content": "Go 项目的测试文件应放在和源文件相同的目录",
			"title":   "Go 测试文件组织",
			"tags":    []interface{}{"go", "testing"},
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
		result, err := server.CallTool("save_experience", map[string]interface{}{
			"title": "只有标题",
			"tags":  []interface{}{"test"},
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("缺少必填参数时应返回错误")
		}
	})

	t.Run("save_experience 缺少必填参数 tags 时返回错误", func(t *testing.T) {
		result, err := server.CallTool("save_experience", map[string]interface{}{
			"content": "没有标签的经验",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("缺少 tags 参数时应返回错误")
		}
	})

	t.Run("save_experience tags 为空数组时返回错误", func(t *testing.T) {
		result, err := server.CallTool("save_experience", map[string]interface{}{
			"content": "空标签的经验",
			"tags":    []interface{}{},
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("tags 为空数组时应返回错误")
		}
	})

	t.Run("search_experiences 正常调用", func(t *testing.T) {
		server.CallTool("save_experience", map[string]interface{}{
			"content": "使用 SQLite 做本地存储时，注意单连接模式避免并发锁",
			"title":   "SQLite 使用经验",
			"tags":    []interface{}{"sqlite", "storage"},
		})

		result, err := server.CallTool("search_experiences", map[string]interface{}{
			"query": "SQLite 怎么用",
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})

	t.Run("search_experiences 缺少 query 时返回错误", func(t *testing.T) {
		result, err := server.CallTool("search_experiences", map[string]interface{}{
			"top_k": 5,
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("缺少 query 参数时应返回错误")
		}
	})

	t.Run("get_experience 正常调用", func(t *testing.T) {
		saveResult, _ := server.CallTool("save_experience", map[string]interface{}{
			"content": "用于测试 Get 的经验",
			"tags":    []interface{}{"test"},
		})
		id := saveResult.Content

		result, err := server.CallTool("get_experience", map[string]interface{}{
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
		result, err := server.CallTool("get_experience", map[string]interface{}{
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
		saveResult, _ := server.CallTool("save_experience", map[string]interface{}{
			"content": "即将被删除的经验",
			"tags":    []interface{}{"test"},
		})
		id := saveResult.Content

		result, err := server.CallTool("delete_experience", map[string]interface{}{
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
		result, err := server.CallTool("list_experiences", map[string]interface{}{
			"page":     1,
			"pageSize": 10,
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应返回错误: %s", result.Content)
		}
	})

	t.Run("list_experiences 无参数时使用默认值", func(t *testing.T) {
		result, err := server.CallTool("list_experiences", map[string]interface{}{})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("无参数应使用默认分页，不应报错: %s", result.Content)
		}
	})

	t.Run("save_experience 发现相似内容返回警告", func(t *testing.T) {
		// 先保存一条
		server.CallTool("save_experience", map[string]interface{}{
			"content": "使用 errors.Is 和 errors.As 做错误比较",
			"tags":    []interface{}{"go", "error"},
		})

		// 保存完全相同的内容
		result, err := server.CallTool("save_experience", map[string]interface{}{
			"content": "使用 errors.Is 和 errors.As 做错误比较",
			"tags":    []interface{}{"go", "error"},
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
		server.CallTool("save_experience", map[string]interface{}{
			"content": "使用 sync.Pool 复用对象减少 GC 压力",
			"tags":    []interface{}{"go", "performance"},
		})

		// 保存相同内容，带 force=true
		result, err := server.CallTool("save_experience", map[string]interface{}{
			"content": "使用 sync.Pool 复用对象减少 GC 压力",
			"tags":    []interface{}{"go", "performance"},
			"force":   true,
		})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if result.IsError {
			t.Errorf("force=true 应成功保存: %s", result.Content)
		}
	})

	t.Run("调用不存在的工具返回错误", func(t *testing.T) {
		result, err := server.CallTool("nonexistent_tool", map[string]interface{}{})
		if err != nil {
			t.Fatalf("CallTool() error: %v", err)
		}
		if !result.IsError {
			t.Error("调用不存在的工具应返回错误")
		}
	})

	t.Run("update_experience 正常调用", func(t *testing.T) {
		saveResult, _ := server.CallTool("save_experience", map[string]interface{}{
			"content": "原始经验内容",
			"title":   "原始标题",
			"tags":    []interface{}{"test"},
		})
		id := saveResult.Content

		result, err := server.CallTool("update_experience", map[string]interface{}{
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
