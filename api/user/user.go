package user

import (
	"ginblog/config"
	"ginblog/pkg/session"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Init(db *gorm.DB, r *gin.RouterGroup, sess *session.Store, auth config.AuthConfig) {
	repo := NewUserRepository(db)
	serv := NewUserService(repo, sess)
	handler := NewUserHandler(serv, auth)
	SetRouter(r, handler, sess, auth, repo)
}
