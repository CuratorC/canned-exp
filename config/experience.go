package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("experience", func() map[string]interface{} {
		return map[string]interface{}{
			"mcp_url":     config.Env("EXPERIENCE_MCP_URL", ""),
			"totp_secret": config.Env("AUTH_TOTP_SECRET", ""),
			"session_ttl": config.Env("AUTH_SESSION_TTL", "24h"),
		}
	})
}
