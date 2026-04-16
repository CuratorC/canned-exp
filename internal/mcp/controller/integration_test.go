package controller

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	sdkserver "github.com/mark3labs/mcp-go/server"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"go.uber.org/zap"
)

// --- 集成测试用的 mock embedding ---

type integrationMockEmbedding struct {
	dimensions int
}

func (m *integrationMockEmbedding) Embed(ctx context.Context, texts []string) ([][]float64, error) {
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

func (m *integrationMockEmbedding) Dimensions() int {
	return m.dimensions
}

// --- 初始化 ---

func init() {
	logger.Logger = zap.NewNop()
}

// --- 辅助函数 ---

// setupIntegrationTest 创建完整的 MCP 通信管道
func setupIntegrationTest(t *testing.T) (stdinWriter io.Writer, stdoutReader io.Reader, cancel context.CancelFunc) {
	t.Helper()

	// 创建 Service
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
	repo := repository.NewGormRepo(gormDB, vecStore, emb)
	svc := service.NewExperienceService(repo)

	// 创建 MCP Server（通过 mcpgw 桥接到 SDK）
	sdkMCP := mcpgw.BuildMCPServer("canned-exp", "1.0.0", NewServer(svc))
	stdioServer := sdkserver.NewStdioServer(sdkMCP)

	// 创建 stdio 管道
	sr, sw := io.Pipe() // stdin: 我们写，server 读
	or, ow := io.Pipe() // stdout: server 写，我们读

	ctx, cancelFunc := context.WithCancel(context.Background())

	go func() {
		stdioServer.Listen(ctx, sr, ow)
	}()

	return sw, or, cancelFunc
}

// jsonrpcRequest 标准 JSON-RPC 请求（ID 为 nil 时为通知，不携带 id 字段）
type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int        `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonrpcResponse 标准 JSON-RPC 响应
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// sendRequest 发送带 ID 的 JSON-RPC 请求
func sendRequest(t *testing.T, w io.Writer, id int, method string, params interface{}) {
	t.Helper()
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	fmt.Fprintf(w, "%s\n", data)
}

// sendNotification 发送无 ID 的 JSON-RPC 通知（不期望响应）
func sendNotification(t *testing.T, w io.Writer, method string, params interface{}) {
	t.Helper()
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal notification: %v", err)
	}
	fmt.Fprintf(w, "%s\n", data)
}

// readResponse 读取 JSON-RPC 响应
func readResponse(t *testing.T, r io.Reader) jsonrpcResponse {
	t.Helper()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	if !scanner.Scan() {
		t.Fatal("no response received from MCP server")
	}

	var resp jsonrpcResponse
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v, raw: %s", err, scanner.Text())
	}
	return resp
}

// --- 测试用例 ---

func TestMCPIntegration_Initialize(t *testing.T) {
	stdinW, stdoutR, cancel := setupIntegrationTest(t)
	defer cancel()

	t.Run("initialize 握手成功", func(t *testing.T) {
		sendRequest(t, stdinW, 1, "initialize", map[string]interface{}{
			"protocolVersion": "2025-03-26",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0",
			},
		})

		resp := readResponse(t, stdoutR)
		if resp.ID != 1 {
			t.Errorf("ID = %d, want 1", resp.ID)
		}
		if resp.Error != nil {
			t.Fatalf("initialize error: %s", resp.Error.Message)
		}

		// 验证响应包含 serverInfo
		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)
		if info, ok := result["serverInfo"].(map[string]interface{}); ok {
			if info["name"] != "canned-exp" {
				t.Errorf("server name = %v, want canned-exp", info["name"])
			}
		} else {
			t.Error("响应缺少 serverInfo")
		}
	})
}

func TestMCPIntegration_ToolsList(t *testing.T) {
	stdinW, stdoutR, cancel := setupIntegrationTest(t)
	defer cancel()

	// 先初始化
	sendRequest(t, stdinW, 1, "initialize", map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "test", "version": "1.0"},
	})
	readResponse(t, stdoutR)

	// 发送 initialized 通知（无 ID，不期望响应）
	sendNotification(t, stdinW, "notifications/initialized", nil)

	t.Run("tools/list 返回 6 个工具", func(t *testing.T) {
		sendRequest(t, stdinW, 2, "tools/list", map[string]interface{}{})
		resp := readResponse(t, stdoutR)
		if resp.Error != nil {
			t.Fatalf("tools/list error: %s", resp.Error.Message)
		}

		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)

		tools, ok := result["tools"].([]interface{})
		if !ok {
			t.Fatal("响应缺少 tools 数组")
		}
		if len(tools) != 6 {
			t.Errorf("工具数量 = %d, want 6", len(tools))
		}

		// 验证每个工具都有 name 和 description
		for _, tool := range tools {
			tl := tool.(map[string]interface{})
			if tl["name"] == nil || tl["name"] == "" {
				t.Error("工具缺少 name")
			}
			if tl["description"] == nil || tl["description"] == "" {
				t.Errorf("工具 %v 缺少 description", tl["name"])
			}
		}
	})
}

func TestMCPIntegration_ToolCall(t *testing.T) {
	stdinW, stdoutR, cancel := setupIntegrationTest(t)
	defer cancel()

	// 初始化
	sendRequest(t, stdinW, 1, "initialize", map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "test", "version": "1.0"},
	})
	readResponse(t, stdoutR)
	sendNotification(t, stdinW, "notifications/initialized", nil)

	t.Run("save_experience 存储成功", func(t *testing.T) {
		sendRequest(t, stdinW, 10, "tools/call", map[string]interface{}{
			"name": "save_experience",
			"arguments": map[string]interface{}{
				"content": "Go 测试文件应与源文件同目录",
				"tags": []interface{}{"go", "testing"},
				"title":   "Go 测试组织",
			},
		})
		resp := readResponse(t, stdoutR)
		if resp.Error != nil {
			t.Fatalf("save error: %s", resp.Error.Message)
		}

		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)
		if isError, _ := result["isError"].(bool); isError {
			t.Errorf("不应报错: %v", result["content"])
		}
	})

	t.Run("search_experiences 搜索成功", func(t *testing.T) {
		sendRequest(t, stdinW, 11, "tools/call", map[string]interface{}{
			"name": "search_experiences",
			"arguments": map[string]interface{}{
				"query": "Go 测试",
			},
		})
		resp := readResponse(t, stdoutR)
		if resp.Error != nil {
			t.Fatalf("search error: %s", resp.Error.Message)
		}

		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)
		if isError, _ := result["isError"].(bool); isError {
			t.Errorf("不应报错: %v", result["content"])
		}
	})

	t.Run("获取不存在的经验返回 IsError", func(t *testing.T) {
		sendRequest(t, stdinW, 12, "tools/call", map[string]interface{}{
			"name": "get_experience",
			"arguments": map[string]interface{}{
				"id": "nonexistent-id",
			},
		})
		resp := readResponse(t, stdoutR)
		if resp.Error != nil {
			// 协议级错误不应该发生
			t.Fatalf("不应返回协议错误: %s", resp.Error.Message)
		}

		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)
		if isError, _ := result["isError"].(bool); !isError {
			t.Error("获取不存在的 ID 应返回 IsError=true")
		}
	})

	t.Run("调用不存在的工具返回协议错误", func(t *testing.T) {
		sendRequest(t, stdinW, 13, "tools/call", map[string]interface{}{
			"name":      "nonexistent_tool",
			"arguments": map[string]interface{}{},
		})
		resp := readResponse(t, stdoutR)
		if resp.Error == nil {
			t.Error("调用不存在的工具应返回协议错误")
		}
	})

	t.Run("save 相同内容触发去重警告", func(t *testing.T) {
		// 保存原始经验
		sendRequest(t, stdinW, 20, "tools/call", map[string]interface{}{
			"name": "save_experience",
			"arguments": map[string]interface{}{
				"content": "使用 context.WithTimeout 控制请求超时",
				"tags": []interface{}{"go", "context"},
			},
		})
		resp := readResponse(t, stdoutR)
		if resp.Error != nil {
			t.Fatalf("首次保存不应报错: %s", resp.Error.Message)
		}

		// 保存完全相同的内容
		sendRequest(t, stdinW, 21, "tools/call", map[string]interface{}{
			"name": "save_experience",
			"arguments": map[string]interface{}{
				"content": "使用 context.WithTimeout 控制请求超时",
				"tags": []interface{}{"go", "context"},
			},
		})
		resp = readResponse(t, stdoutR)
		if resp.Error != nil {
			t.Fatalf("去重检查不应返回协议错误: %s", resp.Error.Message)
		}

		var result map[string]interface{}
		json.Unmarshal(resp.Result, &result)
		if isError, _ := result["isError"].(bool); isError {
			t.Error("去重检测不应返回 isError=true")
		}
		contentArr, _ := result["content"].([]interface{})
		if len(contentArr) == 0 {
			t.Fatal("响应缺少 content")
		}
		textObj, _ := contentArr[0].(map[string]interface{})
		text, _ := textObj["text"].(string)
		if len(text) < 20 {
			t.Errorf("去重响应应包含详细信息，实际: %s", text)
		}
	})
}

// 确保接口实现
var _ embedding.Provider = (*integrationMockEmbedding)(nil)
