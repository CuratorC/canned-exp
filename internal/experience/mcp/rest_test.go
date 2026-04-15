package mcp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"canned-exp/internal/experience"
	"canned-exp/internal/experience/vectorstore"

	_ "modernc.org/sqlite"
)

// setupRESTTest 创建 REST handler 和内存数据库
func setupRESTTest(t *testing.T) (http.Handler, *experience.Service) {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
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
	handler := NewRESTHandler(svc)
	return handler, svc
}

// doSearch 发送 POST /api/search 请求并返回响应
func doSearch(handler http.Handler, body interface{}) *http.Response {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/search", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Result()
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
func saveTestExp(t *testing.T, svc *experience.Service, content, title string, tags []string) {
	t.Helper()
	exp := &experience.Experience{
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

func TestRESTHandler_Search_Success(t *testing.T) {
	handler, svc := setupRESTTest(t)

	// 先保存几条经验
	saveTestExp(t, svc, "Go 项目的测试文件应放在和源文件相同的目录", "Go 测试组织", []string{"go", "testing"})
	saveTestExp(t, svc, "使用 SQLite 做本地存储时注意并发锁", "SQLite 经验", []string{"sqlite", "storage"})

	resp := doSearch(handler, map[string]interface{}{
		"query":  "Go 测试",
		"top_k": 3,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	result := parseHookOutput(t, resp.Body)

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

func TestRESTHandler_Search_MissingQuery(t *testing.T) {
	handler, _ := setupRESTTest(t)

	resp := doSearch(handler, map[string]interface{}{
		"top_k": 3,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRESTHandler_Search_EmptyResults(t *testing.T) {
	handler, _ := setupRESTTest(t)

	// 不保存任何经验，搜索应返回空 JSON
	resp := doSearch(handler, map[string]interface{}{
		"query":  "完全不存在的查询内容 xyz123",
		"top_k": 3,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	// 无结果时应返回空 JSON（无 hookSpecificOutput）
	if _, hasHook := result["hookSpecificOutput"]; hasHook {
		t.Error("无结果时不应包含 hookSpecificOutput")
	}
}

func TestRESTHandler_Search_TopKParameter(t *testing.T) {
	handler, svc := setupRESTTest(t)

	// 保存 5 条关于 Go 的经验
	for i := 0; i < 5; i++ {
		saveTestExp(t, svc,
			"Go 并发编程经验第"+string(rune('A'+i))+"条，关于 goroutine 和 channel 的使用",
			"Go 并发"+string(rune('A'+i)),
			[]string{"go", "concurrency"},
		)
	}

	resp := doSearch(handler, map[string]interface{}{
		"query":  "Go 并发",
		"top_k":  2,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	result := parseHookOutput(t, resp.Body)
	hookOutput, _ := result["hookSpecificOutput"].(map[string]interface{})
	ctx, _ := hookOutput["additionalContext"].(string)

	// 验证只返回 top_k 条结果
	// 统计 "Go 并发编程经验" 出现次数
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
