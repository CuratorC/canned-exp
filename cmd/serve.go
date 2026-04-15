package cmd

import (
	"canned-exp/bootstrap"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CuratorC/gocanned/config"
	"github.com/CuratorC/gocanned/env"
	"github.com/CuratorC/gocanned/logger"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// StartServe represents the available web sub-command.
var StartServe = &cobra.Command{
	Use:   "serve",
	Short: "Start web server",
	Run:   runWeb,
	Args:  cobra.NoArgs,
}

func runWeb(cmd *cobra.Command, args []string) {

	application, err := bootstrap.NewApp()
	if err != nil {
		logger.ErrorAndExit("Failed to setup app", err)
	}

	// 设置 gin 的运行模式，支持 debug, release, test
	// release 会屏蔽调试信息，官方建议生产环境中使用
	// 非 release 模式 gin 终端打印太多信息，干扰到我们程序中的 Log
	// 故此设置为 release，有特殊情况手动改为 debug 即可
	gin.SetMode(gin.ReleaseMode)

	// gin 实例
	router := gin.New()

	if !env.IsProduction() {
		pprof.Register(router)
	}

	// 初始化路由绑定，将依赖注入到路由层
	bootstrap.SetupRoute(router, application)

	// 打印所有路由
	var fields []zap.Field
	for _, route := range router.Routes() {
		fields = append(fields, zap.String(route.Method, route.Path))
	}
	logger.Info("已注册路由", fields...)

	port := config.GetString("app.port")
	logger.Info("Server started at port " + port)

	// 使用 http.Server 以支持优雅关停
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// 非阻塞启动服务器
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorAndExit("Unable to start server", err)
		}
	}()

	// 监听中断信号，优雅关停
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	logger.Info(fmt.Sprintf("Received signal %v, shutting down...", sig))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}
