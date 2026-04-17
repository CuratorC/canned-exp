package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"canned-exp/internal/model"
	"canned-exp/internal/repository"
	"canned-exp/internal/service"
	"canned-exp/internal/vectorstore"

	_ "canned-exp/internal/database/migrations/main"

	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/logger"
	"github.com/CuratorC/gocanned/migration"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func init() {
	logger.Logger = zap.NewNop()
}

// mockEmbedding 测试用 mock
type mockEmbedding struct {
	dimensions int
}

func (m *mockEmbedding) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	results := make([][]float64, len(texts))
	for i := range texts {
		vec := make([]float64, m.dimensions)
		for j := range vec {
			vec[j] = float64(i + j + 1)
		}
		results[i] = vec
	}
	return results, nil
}

func (m *mockEmbedding) Dimensions() int {
	return m.dimensions
}

// setupSearchTest 创建 Search handler 和内存数据库
func setupSearchTest(t *testing.T) (*service.ExperienceService, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)

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
	repo := repository.NewExperienceGormRepo(gormDB, vecStore, emb)
	svc := service.NewExperienceService(repo)

	r := gin.New()
	r.POST("/api/search", SearchHandler(svc))
	return svc, r
}

// doSearch 发送 POST /api/search 请求并返回响应
func doSearch(router *gin.Engine, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/search", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// parseHookOutput 解析响应体为 Hook 输出格式
func parseHookOutput(t *testing.T, body io.Reader) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.NewDecoder(body).Decode(&result); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	return result
}

// saveTestExp 保存一条测试经验
func saveTestExp(t *testing.T, svc *service.ExperienceService, content, title string, tags []string) {
	t.Helper()
	exp := &model.Experience{
		Content: content,
		Title:   title,
		Tags:    tags,
	}
	_, err := svc.Save(t.Context(), exp, false)
	if err != nil {
		t.Fatalf("save test exp: %v", err)
	}
}

// --- 测试用例 ---

func TestSearchHandler_Search_Success(t *testing.T) {
	svc, router := setupSearchTest(t)

	// 先保存几条经验
	saveTestExp(t, svc, "Go 项目的测试文件应放在和源文件相同的目录", "Go 测试组织", []string{"go", "testing"})
	saveTestExp(t, svc, "使用 SQLite 做本地存储时注意并发锁", "SQLite 经验", []string{"sqlite", "storage"})

	w := doSearch(router, map[string]interface{}{
		"query":  "Go 测试",
		"top_k": 3,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	result := parseHookOutput(t, w.Body)

	// 验证 hookSpecificOutput 结构
	hookOutput, ok := result["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatal("响应缺少 hookSpecificOutput")
	}
	if hookOutput["hookEventName"] != "UserPromptSubmit" {
		t.Errorf("hookEventName = %v, want UserPromptSubmit", hookOutput["hookEventName"])
	}
	ctx, ok := hookOutput["additionalContext"].(string)
	if !ok || ctx == "" {
		t.Fatal("additionalContext 不应为空")
	}
	// 内容应包含经验文本
	if len(ctx) < 20 {
		t.Errorf("additionalContext 过短: %s", ctx)
	}
}

func TestSearchHandler_Search_MissingQuery(t *testing.T) {
	_, router := setupSearchTest(t)

	w := doSearch(router, map[string]interface{}{
		"top_k": 3,
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestSearchHandler_Search_EmptyResults(t *testing.T) {
	_, router := setupSearchTest(t)

	// 不保存任何经验，搜索应返回空 JSON
	w := doSearch(router, map[string]interface{}{
		"query":  "完全不存在的查询内容 xyz123",
		"top_k": 3,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	// 无结果时应返回空 JSON（无 hookSpecificOutput）
	if _, hasHook := result["hookSpecificOutput"]; hasHook {
		t.Error("无结果时不应包含 hookSpecificOutput")
	}
}

func TestSearchHandler_Search_TopKParameter(t *testing.T) {
	svc, router := setupSearchTest(t)

	// 保存 5 条关于 Go 的经验
	for i := 0; i < 5; i++ {
		saveTestExp(t, svc,
			"Go 并发编程经验第"+string(rune('A'+i))+"条，关于 goroutine 和 channel 的使用",
			"Go 并发"+string(rune('A'+i)),
			[]string{"go", "concurrency"},
		)
	}

	w := doSearch(router, map[string]interface{}{
		"query":  "Go 并发",
		"top_k":  2,
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	result := parseHookOutput(t, w.Body)
	hookOutput, _ := result["hookSpecificOutput"].(map[string]interface{})
	ctx, _ := hookOutput["additionalContext"].(string)

	// 验证只返回 top_k 条结果
	// 统计换行数量
	count := 0
	for _, r := range ctx {
		if r == '\n' {
			count++
		}
	}
	// top_k=2，应有 2 行（最后一行可能没有换行，所以 count 应为 1 或 2）
	if count > 2 {
		t.Errorf("返回结果超过 top_k=2: additionalContext 包含过多条目\n%s", ctx)
	}
}
