package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/CuratorC/gocanned/helper"
	"github.com/CuratorC/gocanned/logger"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// truncateBody 截断过长的 body：先尝试 JSON pretty-print，再按完整行截断
func truncateBody(body []byte, max int) string {
	if len(body) == 0 {
		return ""
	}

	display := body
	// 尝试 pretty-print JSON，让每行是一个完整的 key-value
	var indented bytes.Buffer
	if json.Indent(&indented, body, "", "  ") == nil {
		display = indented.Bytes()
	}

	if len(display) <= max {
		return string(display)
	}

	// 在 max 范围内找到最后一个换行符，按完整行截断
	cut := bytes.LastIndex(display[:max], []byte("\n"))
	if cut <= 0 {
		cut = max
	}
	return string(display[:cut]) + fmt.Sprintf("\n... [truncated, total %d bytes]", len(body))
}

// Logger 记录请求日志
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {

		// 获取 response 内容
		w := &responseBodyWriter{body: &bytes.Buffer{}, ResponseWriter: c.Writer}
		c.Writer = w

		// 获取请求数据
		var requestBody []byte
		if c.Request.Body != nil {
			// c.Request.Body 是一个 buffer 对象，只能读取一次
			requestBody, _ = io.ReadAll(c.Request.Body)
			// 读取后，重新赋值 c.Request.Body ，以供后续的其他操作
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 设置开始时间
		start := time.Now()
		c.Next()

		// 开始记录日志的逻辑
		cost := time.Since(start)
		responseStatus := c.Writer.Status()

		logFields := []zap.Field{
			zap.Int("status", responseStatus),
			zap.String("request", c.Request.Method+" "+c.Request.URL.String()),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.String("time", helper.MicrosecondsStr(cost)),
		}

		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
			// 请求的内容
			logFields = append(logFields, zap.String("Request Body", truncateBody(requestBody, 512)))

			// 响应的内容
			logFields = append(logFields, zap.String("Response Body", truncateBody(w.body.Bytes(), 512)))
		}

		if responseStatus > 400 && responseStatus <= 499 {
			// 除了 StatusBadRequest 以外，warning 提示一下，常见的有 403 404，开发时都要注意
			logger.Warn("HTTP Warning "+cast.ToString(responseStatus), logFields...)
		} else if responseStatus >= 500 && responseStatus <= 599 {
			// 除了内部错误，记录 error
			logger.Error("HTTP Error "+cast.ToString(responseStatus), logFields...)
		} else {
			logger.Debug("HTTP Access Log", logFields...)
		}
	}
}
