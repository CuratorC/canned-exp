package bootstrap

import (
	"canned-exp/internal/app"
	"canned-exp/internal/enum"

	ccmd "github.com/CuratorC/gocanned/cmd"
	"github.com/CuratorC/gocanned/cerr"
	"github.com/CuratorC/gocanned/database"
)

func SetupCommand(envSuffix string) (err error) {
	err = SetupConfig(envSuffix)
	if err != nil {
		return cerr.Wrap(err, "failed to SetupConfig")
	}
	err = SetupLogger()
	if err != nil {
		return cerr.Wrap(err, "failed to SetupLogger")
	}

	return nil
}

// NewApp initializes infrastructure (Redis, Database) and returns the App dependency container.
func NewApp() (*app.App, error) {
	application := &app.App{}

	err := SetupRedis()
	if err != nil {
		return nil, cerr.Wrap(err, "failed to SetupRedis")
	}

	db, err := SetupDatabase(enum.DatabaseNameMain)
	if err != nil {
		return nil, cerr.Wrap(err, "failed to SetupDatabase main")
	}
	application.DB = db

	// 设置迁移模块的数据库连接获取函数
	ccmd.SetGetDBFunc(func(dbName string) *database.DB {
		return application.DB
	})

	return application, nil
}
