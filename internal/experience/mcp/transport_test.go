package mcp

import (
	"context"
	"database/sql"
	"math"
	"testing"

	"canned-exp/internal/experience"
	"canned-exp/internal/experience/embedding"
	"canned-exp/internal/experience/vectorstore"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	_ "modernc.org/sqlite"
)

// mockEmbedding 测试用 mock
type transportMockEmbedding struct {
	dimensions int
}

func (m *transportMockEmbedding) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i, text := range texts {
		vec := make([]float64, m.dimensions)
		for _, r := range text {
			idx := int(uint32(r)) % m.dimensions
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
		results[i] = vec
	}
	return results, nil
}

func (m *transportMockEmbedding) Dimensions() int {
	return m.dimensions
}

func setupTransportTestSvc(t *testing.T) *experience.Service {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	vecStore, err := vectorstore.NewSQLiteVecStore(db)
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}
	emb := &transportMockEmbedding{dimensions: 64}
	repo, err := experience.NewSQLiteRepo(db, vecStore, emb)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}
	return experience.NewService(repo)
}

// --- 工具转换测试 ---

func TestTransport_ConvertTool(t *testing.T) {
	svc := setupTransportTestSvc(t)
	ourServer := NewServer(svc)

	t.Run("所有工具都能成功转换为 MCP SDK Tool", func(t *testing.T) {
		for _, td := range ourServer.ListTools() {
			tool := convertTool(td)
			if tool.Name != td.Name {
				t.Errorf("Name = %q, want %q", tool.Name, td.Name)
			}
			if tool.Description == "" {
				t.Errorf("工具 %q 转换后描述为空", td.Name)
			}
			if tool.InputSchema.Type == "" {
				t.Errorf("工具 %q 转换后 InputSchema.Type 为空", td.Name)
			}
		}
	})

	t.Run("save_experience 的 content 和 tags 是必填参数", func(t *testing.T) {
		var saveDef *ToolDefinition
		for i := range ourServer.ListTools() {
			td := ourServer.ListTools()[i]
			if td.Name == "save_experience" {
				saveDef = &td
				break
			}
		}
		if saveDef == nil {
			t.Fatal("未找到 save_experience")
		}

		tool := convertTool(*saveDef)
		hasContent := false
		hasTags := false
		for _, r := range tool.InputSchema.Required {
			if r == "content" {
				hasContent = true
			}
			if r == "tags" {
				hasTags = true
			}
		}
		if !hasContent {
			t.Error("content 应在 required 列表中")
		}
		if !hasTags {
			t.Error("tags 应在 required 列表中")
		}
	})
}

// --- SDK Server 构建测试 ---

func TestTransport_BuildMCPServer(t *testing.T) {
	svc := setupTransportTestSvc(t)

	t.Run("buildMCPServer 不 panic 且返回有效实例", func(t *testing.T) {
		mcpServer := buildMCPServer(svc)
		if mcpServer == nil {
			t.Fatal("buildMCPServer 返回 nil")
		}
	})

	t.Run("StdioServer 可创建", func(t *testing.T) {
		mcpServer := buildMCPServer(svc)
		stdioServer := server.NewStdioServer(mcpServer)
		if stdioServer == nil {
			t.Fatal("NewStdioServer 返回 nil")
		}
	})
}

// --- 桥接调用测试 ---

func TestTransport_HandlerBridge(t *testing.T) {
	svc := setupTransportTestSvc(t)
	ourServer := NewServer(svc)

	t.Run("通过 handler 桥接存储经验", func(t *testing.T) {
		handler := createHandler(ourServer, "save_experience")
		req := mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "save_experience",
				Arguments: map[string]interface{}{
					"content": "通过 handler 桥接测试",
					"title":   "桥接测试",
					"tags":    []interface{}{"test"},
				},
			},
		}

		result, err := handler(context.Background(), req)
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应报错: %v", result.Content)
		}
		if len(result.Content) == 0 {
			t.Error("应返回内容")
		}
	})

	t.Run("通过 handler 桥接搜索经验", func(t *testing.T) {
		// 先存一条
		handler := createHandler(ourServer, "save_experience")
		handler(context.Background(), mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "save_experience",
				Arguments: map[string]interface{}{
					"content": "Redis 缓存策略经验",
					"tags":    []interface{}{"redis", "cache"},
				},
			},
		})

		// 搜索
		searchHandler := createHandler(ourServer, "search_experiences")
		result, err := searchHandler(context.Background(), mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "search_experiences",
				Arguments: map[string]interface{}{
					"query": "Redis",
				},
			},
		})
		if err != nil {
			t.Fatalf("handler error: %v", err)
		}
		if result.IsError {
			t.Errorf("不应报错: %v", result.Content)
		}
	})

	t.Run("handler 错误时正确标记 IsError", func(t *testing.T) {
		handler := createHandler(ourServer, "get_experience")
		result, err := handler(context.Background(), mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "get_experience",
				Arguments: map[string]interface{}{
					"id": "nonexistent",
				},
			},
		})
		if err != nil {
			t.Fatalf("handler 不应返回 go error: %v", err)
		}
		if !result.IsError {
			t.Error("获取不存在的经验应标记 IsError=true")
		}
	})
}

// 确保接口实现
var _ embedding.Provider = (*transportMockEmbedding)(nil)
