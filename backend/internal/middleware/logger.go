package middleware

import (
	"log/slog"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var Logger *slog.Logger

func InitLogger() {
	// Setup structured JSON logger to stdout
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(Logger)
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				Logger.Error("request_failed",
					slog.String("method", method),
					slog.String("path", path),
					slog.String("query", query),
					slog.Int("status", status),
					slog.String("ip", clientIP),
					slog.Duration("latency", latency),
					slog.String("error", e),
				)
			}
		} else {
			Logger.Info("request_success",
				slog.String("method", method),
				slog.String("path", path),
				slog.String("query", query),
				slog.Int("status", status),
				slog.String("ip", clientIP),
				slog.Duration("latency", latency),
			)
		}
	}
}
