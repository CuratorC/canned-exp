package config

import "github.com/CuratorC/gocanned/config"

func init() {
	config.Add("log", func() map[string]interface{} {
		return map[string]interface{}{
				"level":          config.Env("LOG_LEVEL", "debug"), // debug, info, warn, error
				"type":           config.Env("LOG_TYPE", "console"), // console, json
				"filename":       config.Env("LOG_NAME", "storage/logs/app.log"),
				"error_filename": config.Env("LOG_ERROR_NAME", "storage/logs/error.log"),
		}
	})
}
