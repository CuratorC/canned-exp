package controller

import (
	"fmt"

	"canned-exp/internal/http/response"
	"canned-exp/internal/service"

	"github.com/gin-gonic/gin"
)

// searchRequest REST 搜索请求
type searchRequest struct {
	Query string `json:"query" binding:"required"`
	TopK  int    `json:"top_k"`
}

// SearchHandler 处理 POST /api/search
// 直接调用 Service.Search()，返回 Claude Code Hook 兼容的 JSON
func SearchHandler(svc *service.ExperienceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req searchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err)
			return
		}

		results, err := svc.Search(c.Request.Context(), req.Query, req.TopK)
		if err != nil {
			response.Abort500(c, fmt.Sprintf("search error: %v", err))
			return
		}

		// 无结果时返回空 JSON
		if len(results) == 0 {
			response.JSON(c, gin.H{})
			return
		}

		// 格式化为 compact text，适合注入 additionalContext
		ctx := ""
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
			ctx += fmt.Sprintf("%d.%s %s%s (score: %.2f)\n",
				i+1, tags, content, "", r.Score)
		}

		resp := map[string]interface{}{
			"hookSpecificOutput": map[string]interface{}{
				"hookEventName":    "UserPromptSubmit",
				"additionalContext": "[相关经验记忆]\n" + ctx,
			},
		}

		response.JSON(c, resp)
	}
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
