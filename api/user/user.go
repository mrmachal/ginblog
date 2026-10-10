package user

import (
	"ginblog/config"
	"ginblog/pkg/session"

	"github.com/gin-gonic/gin"
)

// Init 不再自己 NewUserRepository——userRepo 由 api.go 建好后注入，
// 保证全进程只有一个实例（article 的鉴权读的就是它）。
func Init(r *gin.RouterGroup, sess *session.Store, auth config.AuthConfig, repo Repository) {
	serv := NewUserService(repo, sess)
	SetRouter(r, NewUserHandler(serv, auth), sess, auth, repo)
}
