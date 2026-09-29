package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("query", query),
			slog.Int("status", status),
			slog.String("latency", latency.String()),
			slog.Int("size", c.Writer.Size()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("user_agent", c.Request.UserAgent()),
			slog.String("errors", c.Errors.String()),
		}

		switch {
		case status >= 500:
			slog.LogAttrs(c.Request.Context(), slog.LevelError, "request", attrs...)
		case status >= 400:
			slog.LogAttrs(c.Request.Context(), slog.LevelWarn, "request", attrs...)
		default:
			slog.LogAttrs(c.Request.Context(), slog.LevelInfo, "request", attrs...)
		}
	}
}
