package controller

import (
	"context"
	"testing"

	"canned-exp/internal/repository"
	"canned-exp/internal/service"
	"canned-exp/internal/renderer"
	_ "canned-exp/internal/database/migrations/main"
	"canned-exp/internal/vectorstore"
	mcpgw "canned-exp/internal/mcp"

	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/logger"
	"github.com/CuratorC/gocanned/migration"
	mcpclient "github.com/mark3labs/mcp-go/client"
	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"go.uber.org/zap"
)

func init() {
	logger.Logger = zap.NewNop()
}

// setupSSETest 创建基于 SSE 传输的测试服务器和客户端
func setupSSETest(t *testing.T) (*mcpclient.Client, func()) {
	t.Helper()

	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open gorm memory sqlite: %v", err)
	}
	dbDB := &database.DB{Gorm: gormDB, Driver: "sqlite"}
	runner := migration.NewRunner("main", dbDB)
	if err := runner.Run(); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	vecStore, err := vectorstore.NewSQLiteVecStore(dbDB.SQL())
	if err != nil {
		t.Fatalf("create vector store: %v", err)
	}
	emb := &integrationMockEmbedding{dimensions: 64}
	repo := repository.NewExperienceGormRepo(gormDB, vecStore, emb)
	svc := service.NewExperienceService(repo)
	agentRepo := repository.NewAgentGormRepo(gormDB)
	agentSvc := service.NewAgentService(agentRepo)

		pRepo := repository.NewPersonalityGormRepo(gormDB)
		pkRepo := repository.NewPersonalityKeyGormRepo(gormDB)
		pSvc := service.NewPersonalityService(pRepo, pkRepo)
		pkSvc := service.NewPersonalityKeyService(pkRepo)
		mRepo := repository.NewMemoryGormRepo(gormDB)
		mSvc := service.NewMemoryService(mRepo)

	mcpServer := mcpgw.BuildMCPServer("canned-exp", "1.0.0", NewServer(svc, agentSvc, pSvc, pkSvc, mSvc, renderer.NewRegistry()))
	ts := server.NewTestServer(mcpServer)

	client, err := mcpclient.NewSSEMCPClient(ts.URL + "/sse")
	if err != nil {
		t.Fatalf("create SSE client: %v", err)
	}

	cleanup := func() {
		client.Close()
		ts.Close()
	}
	return client, cleanup
}

// sseInit 启动 SSE 客户端并初始化连接
func sseInit(t *testing.T, client *mcpclient.Client) {
	t.Helper()
	ctx := context.Background()
	if err := client.Start(ctx); err != nil {
		t.Fatalf("start SSE client: %v", err)
	}
	_, err := client.Initialize(ctx, mcpsdk.InitializeRequest{
		Params: mcpsdk.InitializeParams{
			ProtocolVersion: "2025-03-26",
			Capabilities:    mcpsdk.ClientCapabilities{},
			ClientInfo:      mcpsdk.Implementation{Name: "test", Version: "1.0"},
		},
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
}

// --- 测试用例 ---

func TestSSEIntegration_Handshake(t *testing.T) {
	client, cleanup := setupSSETest(t)
	defer cleanup()
	ctx := context.Background()

	t.Run("SSE 握手 + tools/list", func(t *testing.T) {
		if err := client.Start(ctx); err != nil {
			t.Fatalf("start SSE client: %v", err)
		}
		initResult, err := client.Initialize(ctx, mcpsdk.InitializeRequest{
			Params: mcpsdk.InitializeParams{
				ProtocolVersion: "2025-03-26",
				Capabilities:    mcpsdk.ClientCapabilities{},
				ClientInfo:      mcpsdk.Implementation{Name: "test", Version: "1.0"},
			},
		})
		if err != nil {
			t.Fatalf("initialize error: %v", err)
		}
		if initResult.ServerInfo.Name != "canned-exp" {
			t.Errorf("server name = %s, want canned-exp", initResult.ServerInfo.Name)
		}

		toolsResult, err := client.ListTools(ctx, mcpsdk.ListToolsRequest{})
		if err != nil {
			t.Fatalf("tools/list error: %v", err)
		}
		if len(toolsResult.Tools) != 24 {
			t.Errorf("工具数量 = %d, want 24", len(toolsResult.Tools))
		}
		for _, tool := range toolsResult.Tools {
			if tool.Name == "" || tool.Description == "" {
				t.Errorf("工具 %q 缺少 name 或 description", tool.Name)
			}
		}
	})
}

func TestSSEIntegration_ToolCall(t *testing.T) {
	client, cleanup := setupSSETest(t)
	defer cleanup()
	ctx := context.Background()
	sseInit(t, client)

	t.Run("save + search 完整流程", func(t *testing.T) {
		saveResult, err := client.CallTool(ctx, mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "save_experience",
				Arguments: map[string]interface{}{
					"content": "Go 项目的 vendor 目录应加入 .gitignore",
					"title":   "Go vendor 管理",
					"tags":    []interface{}{"go", "git"},
				},
			},
		})
		if err != nil {
			t.Fatalf("save error: %v", err)
		}
		if saveResult.IsError {
			t.Errorf("不应报错: %v", saveResult.Content)
		}

		searchResult, err := client.CallTool(ctx, mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "search_experiences",
				Arguments: map[string]interface{}{
					"query": "Go 项目",
				},
			},
		})
		if err != nil {
			t.Fatalf("search error: %v", err)
		}
		if searchResult.IsError {
			t.Errorf("搜索不应报错: %v", searchResult.Content)
		}
	})

	t.Run("save 去重检测", func(t *testing.T) {
		// 保存原始
		client.CallTool(ctx, mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "save_experience",
				Arguments: map[string]interface{}{
					"content": "使用 sync.Once 保证单例初始化只执行一次",
					"tags":    []interface{}{"go", "concurrency"},
				},
			},
		})

		// 保存相同内容
		result, err := client.CallTool(ctx, mcpsdk.CallToolRequest{
			Params: mcpsdk.CallToolParams{
				Name: "save_experience",
				Arguments: map[string]interface{}{
					"content": "使用 sync.Once 保证单例初始化只执行一次",
					"tags":    []interface{}{"go", "concurrency"},
				},
			},
		})
		if err != nil {
			t.Fatalf("dedup save error: %v", err)
		}
		if result.IsError {
			t.Error("去重检测不应返回 isError")
		}
		if len(result.Content) == 0 {
			t.Fatal("响应缺少 content")
		}
		// 从 Content 中提取 text
		textContent, ok := result.Content[0].(mcpsdk.TextContent)
		if !ok {
			t.Fatalf("响应 content 不是 TextContent: %T", result.Content[0])
		}
		if len(textContent.Text) < 20 {
			t.Errorf("去重响应应包含详细信息，实际: %s", textContent.Text)
		}
	})
}
