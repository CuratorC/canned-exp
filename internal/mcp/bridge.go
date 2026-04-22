package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ConvertTool 将 ToolDefinition 转换为 MCP SDK 的 Tool
func ConvertTool(td ToolDefinition) mcpsdk.Tool {
	opts := []mcpsdk.ToolOption{
		mcpsdk.WithDescription(td.Description),
		mcpsdk.WithReadOnlyHintAnnotation(td.ReadOnly),
		mcpsdk.WithDestructiveHintAnnotation(td.Destructive),
	}

	for _, p := range td.Params {
		switch p.Type {
		case "string":
			opts = append(opts, mcpsdk.WithString(p.Name,
				toParamOptions(p)...,
			))
		case "number":
			opts = append(opts, mcpsdk.WithNumber(p.Name,
				toParamOptions(p)...,
			))
		case "array":
			opts = append(opts, mcpsdk.WithArray(p.Name,
				toParamOptions(p)...,
			))
		case "boolean":
			opts = append(opts, mcpsdk.WithBoolean(p.Name,
				toParamOptions(p)...,
			))
		}
	}

	return mcpsdk.NewTool(td.Name, opts...)
}

func toParamOptions(p ParamDef) []mcpsdk.PropertyOption {
	opts := []mcpsdk.PropertyOption{
		mcpsdk.Description(p.Description),
	}
	if p.Required {
		opts = append(opts, mcpsdk.Required())
	}
	return opts
}

// createHandler 创建 MCP SDK 格式的工具处理函数
func createHandler(provider ToolProvider, toolName string) func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	return func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args := req.GetArguments()
		if args == nil {
			args = make(map[string]interface{})
		}

		result, err := provider.CallTool(ctx, toolName, args)
		if err != nil {
			return &mcpsdk.CallToolResult{
				Content: []mcpsdk.Content{
					mcpsdk.TextContent{Type: "text", Text: fmt.Sprintf("Error: %v", err)},
				},
				IsError: true,
			}, nil
		}

		return &mcpsdk.CallToolResult{
			Content: []mcpsdk.Content{
				mcpsdk.TextContent{Type: "text", Text: result.Content},
			},
			IsError: result.IsError,
		}, nil
	}
}

// BuildMCPServer 创建基于 mark3labs/mcp-go 的 MCP Server
// 将 ToolProvider 桥接到 MCP SDK
func BuildMCPServer(name, version string, provider ToolProvider) *server.MCPServer {
	mcpServer := server.NewMCPServer(
		name,
		version,
		server.WithToolCapabilities(false),
	)

	// 注册所有工具
	for _, toolDef := range provider.ListTools() {
		tool := ConvertTool(toolDef)
		handler := createHandler(provider, toolDef.Name)
		mcpServer.AddTool(tool, handler)
	}

	return mcpServer
}

// NewStdioServer 创建基于 Stdio 传输的 MCP Server（供 CLI 使用）
func NewStdioServer(name, version string, provider ToolProvider) *server.MCPServer {
	return BuildMCPServer(name, version, provider)
}

// NewSSEServer 创建基于 SSE 传输的 MCP Server
func NewSSEServer(name, version string, provider ToolProvider) *server.SSEServer {
	mcpServer := BuildMCPServer(name, version, provider)
	return server.NewSSEServer(mcpServer)
}

// NewStreamableHTTPServer 创建基于 Streamable HTTP 传输的 MCP Server
func NewStreamableHTTPServer(name, version string, provider ToolProvider) *server.StreamableHTTPServer {
	mcpServer := BuildMCPServer(name, version, provider)
	return server.NewStreamableHTTPServer(mcpServer)
}
