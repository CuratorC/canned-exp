package bootstrap

import "github.com/CuratorC/gocanned/logger"

func SetupLogger() (err error) {
	return logger.SetupLogger("log")
}
