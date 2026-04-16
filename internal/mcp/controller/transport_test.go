package controller

import (
	"context"
	"math"
	"testing"

	"canned-exp/internal/embedding"
	_ "canned-exp/internal/database/migrations/main"
	"canned-exp/internal/repository"
	"canned-exp/internal/service"
	"canned-exp/internal/vectorstore"
	mcpgw "canned-exp/internal/mcp"

	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/logger"
	"github.com/CuratorC/gocanned/migration"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"go.uber.org/zap"
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

func init() {
	logger.Logger = zap.NewNop()
}

func setupTransportTestSvc(t *testing.T) *service.ExperienceService {
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
	emb := &transportMockEmbedding{dimensions: 64}
	repo := repository.NewGormRepo(gormDB, vecStore, emb)
	return service.NewExperienceService(repo)
}

// --- 工具转换测试 ---

func TestTransport_ConvertTool(t *testing.T) {
	svc := setupTransportTestSvc(t)
	ourServer := NewServer(svc)

	t.Run("所有工具都能成功转换为 MCP SDK Tool", func(t *testing.T) {
		for _, td := range ourServer.ListTools() {
			tool := mcpgw.ConvertTool(td)
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

		tool := mcpgw.ConvertTool(*saveDef)
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

	t.Run("BuildMCPServer 不 panic 且返回有效实例", func(t *testing.T) {
		expServer := NewServer(svc)
		mcpServer := mcpgw.BuildMCPServer("canned-exp", "1.0.0", expServer)
		if mcpServer == nil {
			t.Fatal("BuildMCPServer 返回 nil")
		}
	})

	t.Run("NewStdioServer 可创建", func(t *testing.T) {
		expServer := NewServer(svc)
		stdioServer := mcpgw.NewStdioServer("canned-exp", "1.0.0", expServer)
		if stdioServer == nil {
			t.Fatal("NewStdioServer 返回 nil")
		}
	})
}

// 确保接口实现
var _ embedding.Provider = (*transportMockEmbedding)(nil)
var _ mcpgw.ToolProvider = (*Server)(nil)
