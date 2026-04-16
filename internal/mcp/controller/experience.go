package controller

import (
	"context"
	"fmt"
	"strconv"

	"canned-exp/internal/model"
	"canned-exp/internal/mcp"
	"canned-exp/internal/service"

	"github.com/bytedance/sonic"
)

// 类型别名，引用通用 MCP 基础设施中的类型定义
type ToolDefinition = mcp.ToolDefinition
type ParamDef = mcp.ParamDef
type CallToolResult = mcp.CallToolResult

// Server MCP Controller，暴露经验库能力给 AI Agent
type Server struct {
	svc   *service.ExperienceService
	tools []ToolDefinition
}

// NewServer 创建 MCP Controller
func NewServer(svc *service.ExperienceService) *Server {
	s := &Server{svc: svc}
	s.registerTools()
	return s
}

func (s *Server) registerTools() {
	s.tools = []ToolDefinition{
		{
			Name:        "save_experience",
			ReadOnly:    false,
			Destructive: false,
			Description: `将经验持久化存储到个人经验库，这是你的长期记忆。

何时主动保存：
- 发现了非显而易见的解决方案（尤其是调试了很久才找到根因的）
- 学到了项目特有的约定、配置或最佳实践
- 需要使用 workaround 绕过某个问题
- 纠正了自己之前的错误认知
- 用户明确要求记住某件事

内容风格：用祈使句/教学体写，包含「问题背景→根因→解决方案→为什么有效」。
例如："当 X 发生时，做 Y，因为 Z" 而不是 "我试了 X 结果失败了"。

标签规范：3-7 个，优先复合标签（如 go-context、mcp-sse、sqlite-vector），
包含领域（go/python/docker）+ 问题模式（error-handling/concurrency/deployment）。
避免过于宽泛的标签如 bug、fix、tip。`,
			Params: []ParamDef{
				{Name: "content", Type: "string", Required: true, Description: "经验的完整叙述内容"},
				{Name: "title", Type: "string", Required: false, Description: "经验的主题摘要（可选）"},
				{Name: "tags", Type: "array", Required: true, Description: "分类标签（必填，用于归类和检索）"},
				{Name: "source", Type: "string", Required: false, Description: "来源 Agent 或应用标识（可选）"},
				{Name: "force", Type: "boolean", Required: false, Description: "发现相似经验时是否强制保存（默认 false）"},
			},
		},
		{
			Name:        "search_experiences",
			ReadOnly:    true,
			Destructive: false,
			Description: `在个人经验库中进行语义检索。基于向量匹配，即使用词不同也能命中。

何时主动检索：
- 开始一个新任务之前（可能之前踩过类似的坑）
- 遇到错误或异常行为时
- 用户提到具体技术/项目/工具名时
- 感觉"似曾相识"时

检索成本极低，宁多搜不少搜。一次遗漏的搜索可能导致重复犯错。`,
			Params: []ParamDef{
				{Name: "query", Type: "string", Required: true, Description: "搜索查询文本（自然语言描述问题）"},
				{Name: "top_k", Type: "number", Required: false, Description: "返回结果数量上限（默认 5）"},
			},
		},
		{
			Name:        "get_experience",
			ReadOnly:    true,
			Destructive: false,
			Description: `通过 ID 获取某条经验的完整内容。当 search_experiences 返回摘要后需要查看详情时使用。`,
			Params: []ParamDef{
				{Name: "id", Type: "string", Required: true, Description: "经验 ID"},
			},
		},
		{
			Name:        "update_experience",
			ReadOnly:    false,
			Destructive: false,
			Description: `更新已有经验。何时更新：已有经验不完整或过时、发现了更好的解决方案。会重新生成向量以保持检索准确性。`,
			Params: []ParamDef{
				{Name: "id", Type: "string", Required: true, Description: "要更新的经验 ID"},
				{Name: "content", Type: "string", Required: false, Description: "更新后的内容"},
				{Name: "title", Type: "string", Required: false, Description: "更新后的标题"},
				{Name: "tags", Type: "array", Required: false, Description: "更新后的标签"},
			},
		},
		{
			Name:        "delete_experience",
			ReadOnly:    false,
			Destructive: true,
			Description: `删除经验。当经验事实错误或已完全过时不再适用时调用。`,
			Params: []ParamDef{
				{Name: "id", Type: "string", Required: true, Description: "要删除的经验 ID"},
			},
		},
		{
			Name:        "list_experiences",
			ReadOnly:    true,
			Destructive: false,
			Description: `分页列出所有经验。用于浏览管理或当语义搜索未能找到期望结果时。`,
			Params: []ParamDef{
				{Name: "page", Type: "number", Required: false, Description: "页码（默认 1）"},
				{Name: "pageSize", Type: "number", Required: false, Description: "每页数量（默认 20）"},
			},
		},
	}
}

// ListTools 返回所有已注册的工具定义
func (s *Server) ListTools() []ToolDefinition {
	return s.tools
}

// CallTool 调用指定工具
func (s *Server) CallTool(ctx context.Context, name string, params map[string]interface{}) (CallToolResult, error) {
	switch name {
	case "save_experience":
		return s.handleSave(ctx, params)
	case "search_experiences":
		return s.handleSearch(ctx, params)
	case "get_experience":
		return s.handleGet(ctx, params)
	case "update_experience":
		return s.handleUpdate(ctx, params)
	case "delete_experience":
		return s.handleDelete(ctx, params)
	case "list_experiences":
		return s.handleList(ctx, params)
	default:
		return CallToolResult{IsError: true, Content: fmt.Sprintf("unknown tool: %s", name)}, nil
	}
}

func (s *Server) handleSave(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	content, ok := params["content"].(string)
	if !ok || content == "" {
		return CallToolResult{IsError: true, Content: "parameter 'content' is required"}, nil
	}

	exp := &model.Experience{
		Content: content,
	}

	if title, ok := params["title"].(string); ok {
		exp.Title = title
	}
	if tags, ok := params["tags"].([]interface{}); ok && len(tags) > 0 {
		for _, t := range tags {
			if str, ok := t.(string); ok {
				exp.Tags = append(exp.Tags, str)
			}
		}
	} else {
		return CallToolResult{IsError: true, Content: "parameter 'tags' is required and must not be empty"}, nil
	}
	if source, ok := params["source"].(string); ok {
		exp.Source = source
	}

	force := false
	if f, ok := params["force"].(bool); ok {
		force = f
	}

	result, err := s.svc.Save(ctx, exp, force)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	if !result.Saved {
		data, _ := sonic.MarshalIndent(map[string]interface{}{
			"message":             "发现相似经验，未保存。使用 force=true 可强制保存。",
			"similar_experiences": result.SimilarExperiences,
		}, "", "  ")
		return CallToolResult{Content: string(data)}, nil
	}

	return CallToolResult{Content: result.ID}, nil
}

func (s *Server) handleSearch(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return CallToolResult{IsError: true, Content: "parameter 'query' is required"}, nil
	}

	topK := 0
	if topKVal, ok := params["top_k"]; ok {
		switch v := topKVal.(type) {
		case float64:
			topK = int(v)
		case int:
			topK = v
		case string:
			topK, _ = strconv.Atoi(v)
		}
	}

	results, err := s.svc.Search(ctx, query, topK)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	data, _ := sonic.MarshalIndent(results, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

func (s *Server) handleGet(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id, ok := params["id"].(string)
	if !ok || id == "" {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	exp, err := s.svc.Get(ctx, id)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	data, _ := sonic.MarshalIndent(exp, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

func (s *Server) handleUpdate(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id, ok := params["id"].(string)
	if !ok || id == "" {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	exp, err := s.svc.Get(ctx, id)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	if title, ok := params["title"].(string); ok {
		exp.Title = title
	}
	if content, ok := params["content"].(string); ok {
		exp.Content = content
	}
	if tags, ok := params["tags"].([]interface{}); ok {
		exp.Tags = nil
		for _, t := range tags {
			if str, ok := t.(string); ok {
				exp.Tags = append(exp.Tags, str)
			}
		}
	}

	if err := s.svc.Update(ctx, exp); err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: "updated"}, nil
}

func (s *Server) handleDelete(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id, ok := params["id"].(string)
	if !ok || id == "" {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	if err := s.svc.Delete(ctx, id); err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: "deleted"}, nil
}

func (s *Server) handleList(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	page := 0
	pageSize := 0

	if pageVal, ok := params["page"]; ok {
		page = toInt(pageVal)
	}
	if pageSizeVal, ok := params["pageSize"]; ok {
		pageSize = toInt(pageSizeVal)
	}

	experiences, total, err := s.svc.List(ctx, page, pageSize)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	result := map[string]interface{}{
		"total":   total,
		"results": experiences,
	}
	data, _ := sonic.MarshalIndent(result, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		n, _ := strconv.Atoi(val)
		return n
	default:
		return 0
	}
}
