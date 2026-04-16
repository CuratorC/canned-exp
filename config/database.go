package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("database", func() map[string]interface{} {
		return map[string]interface{}{
				// 主数据库
				"main": map[string]interface{}{
					"driver":            config.Env("DB_DRIVER", "sqlite"),
					"host":              config.Env("DB_HOST", "127.0.0.1"),
					"port":              config.Env("DB_PORT", 3306),
					"user":              config.Env("DB_USER", "root"),
					"password":          config.Env("DB_PASSWORD", ""),
					"name":              config.Env("DB_NAME", "storage/experience.db"),
					"dsn":               config.Env("DB_DSN", ""),
					"charset":           config.Env("DB_CHARSET", "utf8mb4"),
					"max_open_conns":    config.Env("DB_MAX_OPEN_CONNS", 20),
					"max_idle_conns":    config.Env("DB_MAX_IDLE_CONNS", 10),
					"conn_max_lifetime": config.Env("DB_CONN_MAX_LIFETIME", 3600),
				},
		}
	})
}
