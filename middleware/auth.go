package middleware

import (
	"context"
	"errors"
	"ginblog/config"
	"ginblog/model"
	"ginblog/pkg/response"
	"ginblog/pkg/session"
	"net/http"

	"github.com/gin-gonic/gin"
)

// KeyUserID 认证中间件写入、handler 读取的 context key。
// 用包级 const 而非各处手写字符串，避免拼成 "user_id" / "userid" 两种 key。
const KeyUserID = "user_id"

// SessionStore 中间件对会话存储的最小依赖，*session.Store 天然实现。
// 定义在消费方：middleware 不依赖 pkg/session 的具体类型，测试可注入内存实现。
type SessionStore interface {
	Get(c context.Context, token string) (uint, error)
}

// RequireAuth 强制登录：读 cookie 里的 token → 查 Redis → 把用户 ID 写入 context。
// 校验失败直接 401 短路（c.Abort），后续 handler 无需再判 userID 是否为空。
// 位置：注册在 Logger 内层、业务 handler 外层，401 响应同样会进访问日志。
func RequireAuth(store SessionStore, auth config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(auth.CookieName)
		if err != nil || token == "" {
			abortUnauthorized(c)
			return
		}

		userID, err := store.Get(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, session.ErrNotFound) {
				// token 过期或已被踢下线：凭证无效
				abortUnauthorized(c)
				return
			}
			// Redis 自身故障不是用户的错，返回 500 让前端稍后重试；
			// 误报 401 会让前端以为登录态失效而踢用户回登录页
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError,
				response.Fail(response.CodeInternal, response.MessageOf(response.CodeInternal)))
			return
		}

		c.Set(KeyUserID, userID)
		c.Next()
	}
}

type UserReader interface {
	GetByID(c context.Context, userID uint) (*model.UserInfo, error)
}

func RequireAdmin(users UserReader, roles ...model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint(KeyUserID)
		u, err := users.GetByID(c.Request.Context(), userID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Fail(response.CodeInternal, response.MessageOf(response.CodeInternal)))
			return
		}
		for _, r := range roles {
			if u.Role == r {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, response.Fail(response.CodeForbidden, response.MessageOf(response.CodeForbidden)))
	}
}

// OptionalAuth 可选登录：有合法登录态就写入 userID，没有（或已过期）也放行。
// 用于「登录了就展示个性化内容、没登录展示游客视图」的接口。
// handler 里用 c.GetUint(KeyUserID) == 0 区分游客。
func OptionalAuth(store SessionStore, auth config.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token, err := c.Cookie(auth.CookieName); err == nil && token != "" {
			if userID, err := store.Get(c.Request.Context(), token); err == nil {
				c.Set(KeyUserID, userID)
			}
		}
		c.Next()
	}
}

func abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized,
		response.Fail(response.CodeUnauthorized, response.MessageOf(response.CodeUnauthorized)))
}
