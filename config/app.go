package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("app", func() map[string]interface{} {
		return map[string]interface{}{
				"name":     config.Env("APP_NAME", "SET_YOUR_APP_NAME"),
				"env":      config.Env("APP_ENV", "prod"), // local, lan, dev, pre, prod
				"debug":    config.Env("APP_DEBUG", false),
				"port":     config.Env("APP_PORT", "3000"),
				"url":      config.Env("APP_URL", "http://localhost:3000"),
				"timezone": config.Env("TIMEZONE", "Asia/Shanghai"),
		}
	})
}
