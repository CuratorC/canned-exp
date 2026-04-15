package bootstrap

import (
	middlewares "canned-exp/internal/http/middleware"
	"canned-exp/internal/route"
	"net/http"
	"strings"

	"canned-exp/internal/app"
	"github.com/gin-gonic/gin"
)

func SetupRoute(router *gin.Engine, application *app.App) {
	// 注册全局中间件
	registerGlobalMiddleWare(router)

	// 注册 API 路由
	route.RegisterAPIRoutes(router, application)

	// 配置 404 路由
	setup404Handler(router)
}

func registerGlobalMiddleWare(router *gin.Engine) {
	router.Use(
		middlewares.Logger(),
		middlewares.Recovery(),
	)
}

func setup404Handler(router *gin.Engine) {
	// 处理 404 请求
	router.NoRoute(func(c *gin.Context) {
		// 获取标头信息的 Accept 信息
		acceptString := c.Request.Header.Get("Accept")
		if strings.Contains(acceptString, "text/html") {
			// 如果是 HTML 的话
			c.String(http.StatusNotFound, "404 page not found")
		} else {
			c.JSON(http.StatusNotFound, gin.H{
				"error_code":    404,
				"error_message": "路由未定义",
			})
		}
	})
}
