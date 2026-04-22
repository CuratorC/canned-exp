package bootstrap

import (
	"time"

	"canned-exp/internal/app"
	"canned-exp/internal/auth"
	"canned-exp/internal/enum"

	_ "canned-exp/internal/database/migrations/main"

	ccmd "github.com/CuratorC/gocanned/cmd"
	"github.com/CuratorC/gocanned/cerr"
	"github.com/CuratorC/gocanned/config"
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

// NewApp initializes infrastructure (Database, ExperienceService, Auth) and returns the App dependency container.
func NewApp() (*app.App, error) {
	application := &app.App{}

	db, err := SetupDatabase(enum.DatabaseNameMain)
	if err != nil {
		return nil, cerr.Wrap(err, "failed to SetupDatabase main")
	}
	application.DB = db

	// 设置迁移模块的数据库连接获取函数
	ccmd.SetGetDBFunc(func(dbName string) *database.DB {
		return application.DB
	})

	// 组装经验库依赖（db.Gorm 提供 GORM 连接，db.SQL() 提供 *sql.DB）
	svc, err := SetupExperience(db)
	if err != nil {
		return nil, cerr.Wrap(err, "failed to setup experience service")
	}
	application.ExperienceService = svc

		// 组装 Agent 管理依赖
		application.AgentService = SetupAgentService(db)

		// 组装 Personality 管理依赖
		application.PersonalityService, application.PersonalityKeyService = SetupPersonalityServices(db)

		// 组装 FrameworkMapping 依赖
		application.FrameworkMappingService = SetupFrameworkMappingService(db)

		// 组装 Memory 依赖
		application.MemoryService = SetupMemoryService(db)

	// 组装认证依赖
	application.Auth = auth.NewAuth(auth.Config{
		TOTPSecret: config.GetString("experience.totp_secret"),
		SessionTTL: parseDuration(config.GetString("experience.session_ttl"), 24*time.Hour),
	})

	// 外部可访问 URL（OAuth metadata 需要）
	application.BaseURL = config.GetString("experience.mcp_url")

	return application, nil
}

// parseDuration 解析时间字符串，失败时返回默认值
func parseDuration(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}
