package bootstrap

import (
	"canned-exp/internal/app"
	"canned-exp/internal/proxy"

	"github.com/CuratorC/gocanned/config"
)

// SetupProxy 初始化代理服务
func SetupProxy(application *app.App) {
	enabled := config.GetBool("proxy.enabled")
	application.ProxyEnabled = enabled
	if !enabled {
		return
	}

	application.ProxyService = proxy.NewProxyService(proxy.Config{
		BaseURL:         config.GetString("proxy.base_url"),
		APIKey:          config.GetString("proxy.api_key"),
		ModelMap:        config.GetString("proxy.model_map"),
		DefaultModel:    config.GetString("proxy.default_model"),
		MaxOutputTokens: config.GetInt("proxy.max_output_tokens"),
	})
}
