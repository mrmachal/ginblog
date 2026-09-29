package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// Recovery 捕获 handler 中的 panic：记录错误日志（含堆栈）并返回 500
//
// 注意注册顺序：应放在 Logger 之后（即 Logger 在外层），
// 这样 panic 被恢复后执行流会返回 Logger，访问日志不会丢失
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.ErrorContext(c.Request.Context(), "panic recovered",
					slog.Any("error", rec),
					slog.String("method", c.Request.Method),
					slog.String("path", c.Request.URL.Path),
					slog.String("stack", string(debug.Stack())),
				)
				// 附加到 gin 的错误集合，外层 Logger 会把它写进访问日志的 errors 字段
				_ = c.Error(fmt.Errorf("panic: %v", rec))
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		c.Next()
	}
}
