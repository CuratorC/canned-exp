package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("redis", func() map[string]interface{} {
		return map[string]interface{}{
				"host":      config.Env("REDIS_HOST", "127.0.0.1"),
				"port":      config.Env("REDIS_PORT", "6379"),
				"password":  config.Env("REDIS_PASSWORD", ""),
				"db":        config.Env("REDIS_DB", 0),
		}
	})
}
