package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("experience", func() map[string]interface{} {
		return map[string]interface{}{
			"mcp_port":    config.Env("EXPERIENCE_MCP_PORT", "3100"),
			"totp_secret": config.Env("EXPERIENCE_TOTP_SECRET", ""),
			"api_key":     config.Env("EXPERIENCE_API_KEY", ""),
			"session_ttl": config.Env("EXPERIENCE_SESSION_TTL", "24h"),
		}
	})
}
