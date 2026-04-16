package mcp

import "context"

// ParamDef 工具参数定义
type ParamDef struct {
	Name        string
	Type        string // "string", "number", "array", "boolean"
	Required    bool
	Description string
}

// ToolDefinition MCP 工具定义
type ToolDefinition struct {
	Name        string
	Description string
	Params      []ParamDef
	ReadOnly    bool // 是否只读工具
	Destructive bool // 是否破坏性工具
}

// RequiredParams 返回必填参数名列表
func (t *ToolDefinition) RequiredParams() []string {
	var required []string
	for _, p := range t.Params {
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return required
}

// CallToolResult 工具调用结果
type CallToolResult struct {
	IsError bool
	Content string
}

// ToolProvider 定义 MCP 工具提供者的接口
// 由业务层（如 mcp/controller.Server）实现
type ToolProvider interface {
	// ListTools 返回所有已注册的工具定义
	ListTools() []ToolDefinition
	// CallTool 调用指定工具
	CallTool(ctx context.Context, name string, params map[string]interface{}) (CallToolResult, error)
}
