package article

import (
	"ginblog/config"
	"ginblog/pkg/session"

	"github.com/gin-gonic/gin"
)

// Init 只负责装配本模块的对象图 + 挂路由。
// 依赖由 api.go 注入：repo 是本模块的仓储，roles 用于鉴权（作者或管理员）。
func Init(r *gin.RouterGroup, sess *session.Store, auth config.AuthConfig, repo Repository, roles RoleProvider) {
	serv := NewService(repo, roles)
	SetRouter(r, NewHandler(serv), sess, auth)
}
