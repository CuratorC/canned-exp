package bootstrap

import (
	_ "canned-exp/config"
	_ "canned-exp/internal/database/migrations/main"

	"github.com/CuratorC/gocanned/config"
)

func SetupConfig(envSuffix string) (err error) {
	return config.SetupConfig(envSuffix)
}
