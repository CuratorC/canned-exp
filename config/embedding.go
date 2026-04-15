package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("embedding", func() map[string]interface{} {
		return map[string]interface{}{
			"provider":   config.Env("EMBEDDING_PROVIDER", "zhipu"),
			"api_key":    config.Env("EMBEDDING_API_KEY", ""),
			"base_url":   config.Env("EMBEDDING_BASE_URL", "https://open.bigmodel.cn/api/paas/v4/embeddings"),
			"model":      config.Env("EMBEDDING_MODEL", "embedding-3"),
			"dimensions": config.Env("EMBEDDING_DIMENSIONS", 2048),
		}
	})
}
