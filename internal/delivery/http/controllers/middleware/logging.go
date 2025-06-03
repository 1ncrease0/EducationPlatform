package middleware

import (
	"SkillForge/pkg/logger"
	"fmt"
	"github.com/gin-gonic/gin"
	"time"
)

func LoggingMiddleware(logger logger.Log) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery
		if rawQuery != "" {
			path = fmt.Sprintf("%s?%s", path, rawQuery)
		}
		status := c.Writer.Status()

		msg := fmt.Sprintf("%s %s", method, path)

		logger.Info(msg,
			"status", status,
			"latency", latency,
			"client_ip", clientIP,
		)

		for _, ginErr := range c.Errors {

			logger.ErrorErr("HTTP request error", ginErr.Err,
				"status", status,
				"method", method,
				"path", path,
			)
		}
	}
}
