package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"canned-exp/internal/experience"

	"github.com/bytedance/sonic"
)

// restRequest REST 搜索请求
type restRequest struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k"`
}

// NewRESTHandler 返回处理 /api/search 的 http.Handler
// POST JSON {"query":"...", "top_k":3}
// 直接调用 Service.Search()，返回 Claude Code Hook 兼容的 JSON
func NewRESTHandler(svc *experience.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req restRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Query == "" {
			http.Error(w, "query is required", http.StatusBadRequest)
			return
		}

		results, err := svc.Search(context.Background(), req.Query, req.TopK)
		if err != nil {
			http.Error(w, fmt.Sprintf("search error: %v", err), http.StatusInternalServerError)
			return
		}

		// 无结果时返回空 JSON
		if len(results) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("{}"))
			return
		}

		// 格式化为 compact text，适合注入 additionalContext
		context := ""
		for i, r := range results {
			title := r.Experience.Title
			if title == "" {
				title = "untitled"
			}
			tags := ""
			if len(r.Experience.Tags) > 0 {
				tags = " [" + joinTags(r.Experience.Tags) + "]"
			}
			content := r.Experience.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			context += fmt.Sprintf("%d.%s %s%s (score: %.2f)\n",
				i+1, tags, content, "", r.Score)
		}

		resp := map[string]interface{}{
			"hookSpecificOutput": map[string]interface{}{
				"hookEventName":   "UserPromptSubmit",
				"additionalContext": "[相关经验记忆]\n" + context,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		data, _ := sonic.Marshal(resp)
		w.Write(data)
	})
}

func joinTags(tags []string) string {
	result := ""
	for i, t := range tags {
		if i > 0 {
			result += "/"
		}
		result += t
	}
	return result
}
