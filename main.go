package main

import (
	"canned-exp/bootstrap"
	"canned-exp/cmd"
	"canned-exp/internal/enum"
	"fmt"
	"os"

	ccmd "github.com/CuratorC/gocanned/cmd"
	"github.com/CuratorC/gocanned/database"
	"github.com/CuratorC/gocanned/logger"
	"github.com/spf13/cobra"
)

func main() {

	// 应用的主入口，默认调用 cmd.CmdServe 命令
	var rootCmd = &cobra.Command{
		Use:   "canned-exp",
		Short: "A simple canned project",
		Long:  `Default will run "serve" command, you can use "-h" flag to see all subcommands`,

		// rootCmd 的所有子命令都会执行以下代码
		PersistentPreRun: func(command *cobra.Command, args []string) {
			var err error
			err = bootstrap.SetupCommand(ccmd.Env)
			if err != nil {
				logger.ErrorAndExit("failed to setup app", err)
			}
		},
	}

	// migrate 命令需要独立的数据库初始化（不走 NewApp）
	// PersistentPreRun 会覆盖父命令的，所以要先调基础 setup
	migrateCmd := ccmd.CmdMigrate()
	migrateCmd.PersistentPreRun = func(command *cobra.Command, args []string) {
		if err := bootstrap.SetupCommand(ccmd.Env); err != nil {
			fmt.Fprintln(os.Stderr, "failed to setup app:", err)
			os.Exit(1)
		}
		db, err := bootstrap.SetupDatabase(enum.DatabaseNameMain)
		if err != nil {
			logger.ErrorAndExit("failed to setup database", err)
		}
		ccmd.SetGetDBFunc(func(dbName string) *database.DB {
			return db
		})
	}

	// 注册子命令
	rootCmd.AddCommand(
		ccmd.CmdInitConfig(),
		migrateCmd,
		cmd.StartServe,
	)

	// 配置默认运行 Web 服务
	ccmd.RegisterDefaultCmd(rootCmd, cmd.StartServe)

	// 注册全局参数，--env
	ccmd.RegisterGlobalFlags(rootCmd)

	// 执行主命令
	if err := rootCmd.Execute(); err != nil {
		logger.ErrorAndExit(fmt.Sprintf("Failed to run app with %v", os.Args), err)
	}
}
