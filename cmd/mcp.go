package cmd

import (
	"fmt"
	"time"

	"canned-exp/bootstrap"
	expmcp "canned-exp/internal/experience/mcp"

	"github.com/CuratorC/gocanned/config"
	"github.com/CuratorC/gocanned/logger"
	"github.com/spf13/cobra"
)

// McpCmd 返回 MCP 命令组
func McpCmd() *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP Server commands",
	}
	mcpCmd.AddCommand(&cobra.Command{
		Use:   "serve",
		Short: "Start MCP Server (SSE + REST)",
		Run:   runMCPServe,
		Args:  cobra.NoArgs,
	})
	return mcpCmd
}

func runMCPServe(cmd *cobra.Command, args []string) {
	// 组装经验库依赖
	svc, err := bootstrap.SetupExperience()
	if err != nil {
		logger.ErrorAndExit("failed to setup experience service", err)
	}

	// 读取认证配置
	totpSecret := config.GetString("experience.totp_secret")
	apiKey := config.GetString("experience.api_key")
	sessionTTL := parseDuration(config.GetString("experience.session_ttl"), 24*time.Hour)

	authConfig := expmcp.AuthConfig{
		TOTPSecret: totpSecret,
		APIKey:     apiKey,
		SessionTTL: sessionTTL,
	}

	// 创建组合服务器（SSE + REST + Auth）
	combinedServer := expmcp.NewCombinedServer(svc, authConfig)

	port := config.GetString("experience.mcp_port")
	if port == "" {
		port = "3100"
	}
	addr := fmt.Sprintf(":%s", port)

	logger.Info(fmt.Sprintf("starting MCP server (SSE + REST + Auth) on %s ...", addr))

	// 启动组合 HTTP 服务，阻塞直到退出
	if err := combinedServer.Start(addr); err != nil {
		logger.ErrorAndExit("MCP server error", err)
	}
}

// parseDuration 解析时间字符串，失败时返回默认值
func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
