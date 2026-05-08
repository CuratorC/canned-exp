package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("proxy", func() map[string]interface{} {
		return map[string]interface{}{
			"enabled":       config.Env("PROXY_ENABLED", false),
			"base_url":      config.Env("PROXY_BASE_URL", "https://api.openai.com/v1"),
			"api_key":       config.Env("PROXY_API_KEY", ""),
			"model_map":     config.Env("PROXY_MODEL_MAP", ""),
			"default_model": config.Env("PROXY_DEFAULT_MODEL", ""),
		}
	})
}
