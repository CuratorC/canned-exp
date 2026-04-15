package main

import (
	"canned-exp/bootstrap"
	"canned-exp/cmd"
	"fmt"
	"os"

	ccmd "github.com/CuratorC/gocanned/cmd"
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

	// 注册子命令
	rootCmd.AddCommand(
		ccmd.CmdInitConfig(),
		ccmd.CmdMigrate(),
		cmd.StartServe,
		cmd.McpCmd(),
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
