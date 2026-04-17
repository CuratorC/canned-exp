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

type ToolDefinition = mcp.ToolDefinition
type ParamDef = mcp.ParamDef
type CallToolResult = mcp.CallToolResult

// Server MCP Controller，暴露经验库能力给 AI Agent
type Server struct {
	svc               *service.ExperienceService
	agentSvc          *service.AgentService
	personalitySvc    *service.PersonalityService
	personalityKeySvc *service.PersonalityKeyService
	tools             []ToolDefinition
}

// NewServer 创建 MCP Controller
func NewServer(svc *service.ExperienceService, agentSvc *service.AgentService, personalitySvc *service.PersonalityService, personalityKeySvc *service.PersonalityKeyService) *Server {
	s := &Server{svc: svc, agentSvc: agentSvc, personalitySvc: personalitySvc, personalityKeySvc: personalityKeySvc}
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
				{Name: "agent_id", Type: "number", Required: false, Description: "所属 Agent ID（可选，用于多 Agent 隔离）"},
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
				{Name: "agent_id", Type: "number", Required: false, Description: "按 Agent ID 过滤（可选）"},
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
				{Name: "agent_id", Type: "number", Required: false, Description: "按 Agent ID 过滤（可选）"},
			},
		},
		{
			Name:        "register_agent",
			ReadOnly:    false,
			Destructive: false,
			Description: "注册一个新的 AI Agent，拥有独立的经验空间。",
			Params: []ParamDef{
				{Name: "name", Type: "string", Required: true, Description: "Agent 名称"},
				{Name: "description", Type: "string", Required: false, Description: "Agent 描述"},
			},
		},
		{
			Name:        "list_agents",
			ReadOnly:    true,
			Destructive: false,
			Description: "分页列出所有已注册的 Agent。",
			Params: []ParamDef{
				{Name: "page", Type: "number", Required: false, Description: "页码（默认 1）"},
				{Name: "pageSize", Type: "number", Required: false, Description: "每页数量（默认 20）"},
			},
		},
		{
			Name:        "get_agent",
			ReadOnly:    true,
			Destructive: false,
			Description: "通过 ID 获取 Agent 信息。",
			Params: []ParamDef{
				{Name: "id", Type: "number", Required: true, Description: "Agent ID"},
			},
		},
		{
			Name:        "update_agent",
			ReadOnly:    false,
			Destructive: false,
			Description: "更新已有 Agent 的名称或描述。",
			Params: []ParamDef{
				{Name: "id", Type: "number", Required: true, Description: "Agent ID"},
				{Name: "name", Type: "string", Required: false, Description: "新名称"},
				{Name: "description", Type: "string", Required: false, Description: "新描述"},
			},
		},
		{
			Name:        "delete_agent",
			ReadOnly:    false,
			Destructive: true,
			Description: "删除一个 Agent。注意：不会级联删除该 Agent 的经验。",
			Params: []ParamDef{
				{Name: "id", Type: "number", Required: true, Description: "Agent ID"},
			},
		},
		// --- Personality 工具 ---
		{
			Name:        "set_personality",
			ReadOnly:    false,
			Destructive: false,
			Description: "为 Agent 设置人格属性。同一 agent_id + key_id 重复设置会更新（Upsert 语义）。",
			Params: []ParamDef{
				{Name: "agent_id", Type: "number", Required: true, Description: "所属 Agent ID"},
				{Name: "key_id", Type: "number", Required: true, Description: "PersonalityKey ID"},
				{Name: "value", Type: "string", Required: true, Description: "属性值"},
				{Name: "type", Type: "string", Required: false, Description: "值类型：string/text/number/boolean（默认 string）"},
			},
		},
		{
			Name:        "get_personality",
			ReadOnly:    true,
			Destructive: false,
			Description: "通过 ID 获取 Personality 记录。",
			Params: []ParamDef{
				{Name: "id", Type: "number", Required: true, Description: "Personality ID"},
			},
		},
		{
			Name:        "update_personality",
			ReadOnly:    false,
			Destructive: false,
			Description: "更新已有 Personality 的 value 或 type。",
			Params: []ParamDef{
				{Name: "id", Type: "number", Required: true, Description: "Personality ID"},
				{Name: "value", Type: "string", Required: false, Description: "新值"},
				{Name: "type", Type: "string", Required: false, Description: "新类型"},
			},
		},
		{
			Name:        "delete_personality",
			ReadOnly:    false,
			Destructive: true,
			Description: "删除一条 Personality 记录。",
			Params: []ParamDef{
				{Name: "id", Type: "number", Required: true, Description: "Personality ID"},
			},
		},
		{
			Name:        "list_personalities",
			ReadOnly:    true,
			Destructive: false,
			Description: "列出指定 Agent 的所有人格属性。",
			Params: []ParamDef{
				{Name: "agent_id", Type: "number", Required: true, Description: "Agent ID"},
				{Name: "page", Type: "number", Required: false, Description: "页码（默认 1）"},
				{Name: "pageSize", Type: "number", Required: false, Description: "每页数量（默认 20）"},
			},
		},
		// --- PersonalityKey 工具 ---
		{
			Name:        "list_personality_keys",
			ReadOnly:    true,
			Destructive: false,
			Description: "列出所有可用的人格属性 Key。",
			Params: []ParamDef{
				{Name: "page", Type: "number", Required: false, Description: "页码（默认 1）"},
				{Name: "pageSize", Type: "number", Required: false, Description: "每页数量（默认 20）"},
			},
		},
		{
			Name:        "register_personality_key",
			ReadOnly:    false,
			Destructive: false,
			Description: "注册一个新的人格属性 Key。",
			Params: []ParamDef{
				{Name: "key_name", Type: "string", Required: true, Description: "属性名"},
				{Name: "description", Type: "string", Required: false, Description: "属性说明"},
			},
		},
	}
}

func (s *Server) ListTools() []ToolDefinition {
	return s.tools
}

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
	case "register_agent":
		return s.handleRegisterAgent(ctx, params)
	case "list_agents":
		return s.handleListAgents(ctx, params)
	case "get_agent":
		return s.handleGetAgent(ctx, params)
	case "update_agent":
		return s.handleUpdateAgent(ctx, params)
	case "delete_agent":
		return s.handleDeleteAgent(ctx, params)
	case "set_personality":
		return s.handleSetPersonality(ctx, params)
	case "get_personality":
		return s.handleGetPersonality(ctx, params)
	case "update_personality":
		return s.handleUpdatePersonality(ctx, params)
	case "delete_personality":
		return s.handleDeletePersonality(ctx, params)
	case "list_personalities":
		return s.handleListPersonalities(ctx, params)
	case "list_personality_keys":
		return s.handleListPersonalityKeys(ctx, params)
	case "register_personality_key":
		return s.handleRegisterPersonalityKey(ctx, params)
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
	if agentID := toUint(params["agent_id"]); agentID != 0 {
		exp.AgentID = agentID
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
		topK = toInt(topKVal)
	}

	agentID := toUint(params["agent_id"])

	results, err := s.svc.Search(ctx, query, agentID, topK)
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

	agentID := toUint(params["agent_id"])

	experiences, total, err := s.svc.List(ctx, agentID, page, pageSize)
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

// --- Agent 工具 Handler ---

func (s *Server) handleRegisterAgent(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	name, ok := params["name"].(string)
	if !ok || name == "" {
		return CallToolResult{IsError: true, Content: "parameter 'name' is required"}, nil
	}

	agent := &model.Agent{
		Name: name,
	}
	if desc, ok := params["description"].(string); ok {
		agent.Description = desc
	}

	id, err := s.agentSvc.Save(ctx, agent)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: fmt.Sprintf("%d", id)}, nil
}

func (s *Server) handleListAgents(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	page := 0
	pageSize := 0
	if pageVal, ok := params["page"]; ok {
		page = toInt(pageVal)
	}
	if pageSizeVal, ok := params["pageSize"]; ok {
		pageSize = toInt(pageSizeVal)
	}

	agents, total, err := s.agentSvc.List(ctx, page, pageSize)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	result := map[string]interface{}{
		"total":   total,
		"results": agents,
	}
	data, _ := sonic.MarshalIndent(result, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

func (s *Server) handleGetAgent(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id := toUint(params["id"])
	if id == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	agent, err := s.agentSvc.Get(ctx, id)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	data, _ := sonic.MarshalIndent(agent, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

func (s *Server) handleUpdateAgent(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id := toUint(params["id"])
	if id == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	agent, err := s.agentSvc.Get(ctx, id)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	if name, ok := params["name"].(string); ok {
		agent.Name = name
	}
	if desc, ok := params["description"].(string); ok {
		agent.Description = desc
	}

	if err := s.agentSvc.Update(ctx, agent); err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: "updated"}, nil
}

func (s *Server) handleDeleteAgent(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id := toUint(params["id"])
	if id == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	if err := s.agentSvc.Delete(ctx, id); err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: "deleted"}, nil
}

// --- Personality 工具 Handler ---

func (s *Server) handleSetPersonality(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	agentID := toUint(params["agent_id"])
	if agentID == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'agent_id' is required"}, nil
	}
	keyID := toUint(params["key_id"])
	if keyID == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'key_id' is required"}, nil
	}
	value, _ := params["value"].(string)
	if value == "" {
		return CallToolResult{IsError: true, Content: "parameter 'value' is required"}, nil
	}

	p := &model.Personality{
		AgentID: agentID,
		KeyID:   keyID,
		Value:   value,
	}
	if typ, ok := params["type"].(string); ok {
		p.Type = typ
	}

	id, err := s.personalitySvc.Set(ctx, p)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: fmt.Sprintf("%d", id)}, nil
}

func (s *Server) handleGetPersonality(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id := toUint(params["id"])
	if id == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	p, err := s.personalitySvc.Get(ctx, id)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	data, _ := sonic.MarshalIndent(p, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

func (s *Server) handleUpdatePersonality(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id := toUint(params["id"])
	if id == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	p, err := s.personalitySvc.Get(ctx, id)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	if value, ok := params["value"].(string); ok {
		p.Value = value
	}
	if typ, ok := params["type"].(string); ok {
		p.Type = typ
	}

	if err := s.personalitySvc.Update(ctx, p); err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: "updated"}, nil
}

func (s *Server) handleDeletePersonality(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	id := toUint(params["id"])
	if id == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'id' is required"}, nil
	}

	if err := s.personalitySvc.Delete(ctx, id); err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: "deleted"}, nil
}

func (s *Server) handleListPersonalities(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	agentID := toUint(params["agent_id"])
	if agentID == 0 {
		return CallToolResult{IsError: true, Content: "parameter 'agent_id' is required"}, nil
	}

	page := 0
	pageSize := 0
	if pageVal, ok := params["page"]; ok {
		page = toInt(pageVal)
	}
	if pageSizeVal, ok := params["pageSize"]; ok {
		pageSize = toInt(pageSizeVal)
	}

	personalities, total, err := s.personalitySvc.ListByAgent(ctx, agentID, page, pageSize)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	result := map[string]interface{}{
		"total":   total,
		"results": personalities,
	}
	data, _ := sonic.MarshalIndent(result, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

// --- PersonalityKey 工具 Handler ---

func (s *Server) handleListPersonalityKeys(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	page := 0
	pageSize := 0
	if pageVal, ok := params["page"]; ok {
		page = toInt(pageVal)
	}
	if pageSizeVal, ok := params["pageSize"]; ok {
		pageSize = toInt(pageSizeVal)
	}

	keys, total, err := s.personalityKeySvc.List(ctx, page, pageSize)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	result := map[string]interface{}{
		"total":   total,
		"results": keys,
	}
	data, _ := sonic.MarshalIndent(result, "", "  ")
	return CallToolResult{Content: string(data)}, nil
}

func (s *Server) handleRegisterPersonalityKey(ctx context.Context, params map[string]interface{}) (CallToolResult, error) {
	keyName, ok := params["key_name"].(string)
	if !ok || keyName == "" {
		return CallToolResult{IsError: true, Content: "parameter 'key_name' is required"}, nil
	}

	pk := &model.PersonalityKey{
		KeyName: keyName,
	}
	if desc, ok := params["description"].(string); ok {
		pk.Description = desc
	}

	id, err := s.personalityKeySvc.Save(ctx, pk)
	if err != nil {
		return CallToolResult{IsError: true, Content: err.Error()}, nil
	}

	return CallToolResult{Content: fmt.Sprintf("%d", id)}, nil
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

func toUint(v interface{}) uint {
	switch val := v.(type) {
	case float64:
		return uint(val)
	case int:
		return uint(val)
	case string:
		n, _ := strconv.ParseUint(val, 10, 64)
		return uint(n)
	default:
		return 0
	}
}
