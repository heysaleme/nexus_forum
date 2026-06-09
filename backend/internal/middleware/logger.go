package middleware

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var Logger *slog.Logger

func InitLogger() {
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(Logger)
}

func shouldSkipRequestLog(path string) bool {
	if path == "/health" || path == "/metrics" {
		return true
	}
	if strings.HasPrefix(path, "/uploads/") {
		return true
	}
	if strings.HasPrefix(path, "/api/ws/") {
		return true
	}
	return false
}

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		if shouldSkipRequestLog(path) {
			return
		}

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		clientIP := c.ClientIP()

		attrs := []any{
			slog.String("method", method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", status),
			slog.String("ip", clientIP),
			slog.Duration("latency", latency),
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				Logger.Error("request_failed", append(attrs, slog.String("error", e))...)
			}
			return
		}

		switch {
		case status >= 500:
			Logger.Error("request_error", attrs...)
		case status >= 400:
			Logger.Warn("request_client_error", attrs...)
		case latency > 500*time.Millisecond:
			Logger.Warn("request_slow", attrs...)
		default:
			// Routine successful requests are omitted to reduce log noise.
		}
	}
}
