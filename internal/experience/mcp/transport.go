package mcp

import (
	"context"
	"fmt"
	"net/http"

	"canned-exp/internal/experience"

	mcpsdk "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// buildMCPServer 创建基于 mark3labs/mcp-go 的 MCP Server
// 将我们的业务 Server 桥接到 MCP SDK
func buildMCPServer(svc *experience.Service) *server.MCPServer {
	s := NewServer(svc)

	mcpServer := server.NewMCPServer(
		"canned-exp",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// 注册所有工具
	for _, toolDef := range s.ListTools() {
		tool := convertTool(toolDef)
		handler := createHandler(s, toolDef.Name)
		mcpServer.AddTool(tool, handler)
	}

	return mcpServer
}

// convertTool 将我们的 ToolDefinition 转换为 MCP SDK 的 Tool
func convertTool(td ToolDefinition) mcpsdk.Tool {
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
func createHandler(s *Server, toolName string) func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
	return func(ctx context.Context, req mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		// 将 MCP SDK 的参数转换为 map[string]interface{}
		args := req.GetArguments()
		if args == nil {
			args = make(map[string]interface{})
		}

		// 调用我们的业务 Server
		result, err := s.CallTool(toolName, args)
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

// NewStdioMCP 创建并返回可用的 MCP Server（供 CLI 使用）
func NewStdioMCP(svc *experience.Service) *server.MCPServer {
	return buildMCPServer(svc)
}

// NewSSEMCP 创建基于 SSE 传输的 MCP Server
func NewSSEMCP(svc *experience.Service) *server.SSEServer {
	mcpServer := buildMCPServer(svc)
	return server.NewSSEServer(mcpServer)
}

// CombinedServer 同时提供 SSE（MCP）和 REST（Hook）服务的组合服务器
type CombinedServer struct {
	sseServer   *server.SSEServer
	restHandler http.Handler
	auth        *Auth
	httpServer  *http.Server
}

// NewCombinedServer 创建同时支持 SSE + REST 的组合服务器
func NewCombinedServer(svc *experience.Service, authConfig AuthConfig) *CombinedServer {
	auth := NewAuth(authConfig)
	return &CombinedServer{
		sseServer:   NewSSEMCP(svc),
		restHandler: NewRESTHandler(svc),
		auth:        auth,
	}
}

// Start 在指定地址启动组合服务器
func (cs *CombinedServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/api/auth/login", cs.auth.LoginHandler())
	mux.Handle("/api/auth/revoke", cs.auth.RevokeHandler())
	mux.Handle("/api/search", cs.restHandler)
	mux.Handle("/", cs.sseServer)

	// 用认证中间件包裹 mux
	var handler http.Handler = mux
	handler = cs.auth.Middleware(handler)

	cs.httpServer = &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	return cs.httpServer.ListenAndServe()
}
